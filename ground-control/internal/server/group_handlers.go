package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/container-registry/harbor-satellite/ground-control/internal/database"
	"github.com/container-registry/harbor-satellite/ground-control/internal/models"
	"github.com/container-registry/harbor-satellite/ground-control/internal/utils"
	"github.com/container-registry/harbor-satellite/ground-control/reg/harbor"
	"github.com/gorilla/mux"
)

func (s *Server) syncGroup(ctx context.Context, req *models.StateArtifact) (database.Group, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Could not begin transaction:", err)
		return database.Group{}, err
	}

	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("Error: Failed to rollback transaction for failed process: %v", err)
			}
		}
	}()

	q := s.dbQueries.WithTx(tx)

	projects := utils.GetProjectNames(&req.Artifacts)
	params := database.CreateGroupParams{
		GroupName:   req.Group,
		RegistryUrl: os.Getenv("HARBOR_URL"),
		Projects:    projects,
	}
	result, err := q.CreateGroup(ctx, params)
	if err != nil {
		log.Println("Error creating group:", err)
		return database.Group{}, err
	}

	satellites, err := q.GroupSatelliteList(ctx, result.ID)
	if err != nil {
		log.Println("Error listing group satellites:", err)
		return database.Group{}, err
	}

	for _, satellite := range satellites {
		robotAcc, err := q.GetRobotAccBySatelliteID(ctx, satellite.SatelliteID)
		if err != nil {
			log.Println("Error getting robot account by satellite ID:", err)
			return database.Group{}, err
		}

		_, err = utils.UpdateRobotProjects(ctx, projects, robotAcc.RobotID)
		if err != nil {
			log.Println("Error updating robot projects:", err)
			return database.Group{}, err
		}
	}

	satExist, err := harbor.GetProject(ctx, "satellite")
	if err != nil {
		log.Println("Error checking satellite project existence:", err)
		return database.Group{}, newOperationError(http.StatusBadGateway, fmt.Sprintf("Error: Checking satellite project: %v", err))
	}

	if !satExist {
		_, err := harbor.CreateSatelliteProject(ctx)
		if err != nil {
			log.Println("Error creating satellite project:", err)
			return database.Group{}, newOperationError(http.StatusBadGateway, fmt.Sprintf("Error: creating satellite project: %v", err))
		}
	}

	err = utils.CreateStateArtifact(ctx, req)
	if err != nil {
		log.Println("Error creating state artifact:", err)
		return database.Group{}, err
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Commit failed: %v", err)
		return database.Group{}, newOperationError(http.StatusInternalServerError, "Error: Could not commit transaction")
	}
	committed = true

	return result, nil
}

func (s *Server) getGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	group := vars["group"]

	result, err := s.dbQueries.GetGroupByName(r.Context(), group)
	if err != nil {
		log.Printf("Could not get group: %v", err)
		WriteJSONError(w, "Group not found", http.StatusNotFound)
		return
	}

	WriteJSONResponse(w, http.StatusOK, result)
}

func (s *Server) deleteGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupName := vars["group"]

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		err := &AppError{
			Message: "Error: Failed to start database transaction",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	q := s.dbQueries.WithTx(tx)

	committed := false
	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				log.Printf("Error: Failed to rollback transaction for failed process: %v", err)
				return
			}
		} else if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("Error: Failed to rollback transaction for failed process: %v", err)
				return
			}
		}
	}()

	group, err := q.GetGroupByName(r.Context(), groupName)
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: "Error: Group Not Found",
			Code:    http.StatusNotFound,
		}
		HandleAppError(w, err)
		return
	}

	// Remove the group from all associated satellites
	satellites, err := q.GroupSatelliteList(r.Context(), group.ID)
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: "Error: Failed to list satellites for group",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	for _, satellite := range satellites {
		if err := q.RemoveSatelliteFromGroup(r.Context(), database.RemoveSatelliteFromGroupParams{
			SatelliteID: satellite.SatelliteID,
			GroupID:     group.ID,
		}); err != nil {
			log.Println(err)
			err := &AppError{
				Message: "Error: Failed to remove group from satellite",
				Code:    http.StatusInternalServerError,
			}
			HandleAppError(w, err)
			return
		}

		// Update robot permissions and state artifact for each satellite
		robotAcc, err := q.GetRobotAccBySatelliteID(r.Context(), satellite.SatelliteID)
		if err != nil {
			log.Println(err)
			err := &AppError{
				Message: "Error: Failed to update satellite permissions",
				Code:    http.StatusInternalServerError,
			}
			HandleAppError(w, err)
			return
		}

		// Get remaining groups for this satellite
		groupList, err := q.SatelliteGroupList(r.Context(), satellite.SatelliteID)
		if err != nil {
			log.Println(err)
			err := &AppError{
				Message: "Error: Failed to update satellite state",
				Code:    http.StatusInternalServerError,
			}
			HandleAppError(w, err)
			return
		}

		// Update projects and state artifacts
		var projects []string
		var groupStates []string
		for _, g := range groupList {
			grp, err := q.GetGroupByID(r.Context(), g.GroupID)
			if err != nil {
				log.Println(err)
				err := &AppError{
					Message: "Error: Failed to update satellite state",
					Code:    http.StatusInternalServerError,
				}
				HandleAppError(w, err)
				return
			}
			projects = append(projects, grp.Projects...)
			groupStates = append(groupStates, utils.AssembleGroupState(grp.GroupName))
		}

		// Update robot permissions
		_, err = utils.UpdateRobotProjects(r.Context(), projects, robotAcc.RobotID)
		if err != nil {
			log.Println(err)
			err := &AppError{
				Message: "Error: Failed to update satellite permissions",
				Code:    http.StatusInternalServerError,
			}
			HandleAppError(w, err)
			return
		}

		sat, err := q.GetSatellite(r.Context(), satellite.SatelliteID)
		if err != nil {
			log.Println(err)
			err := &AppError{
				Message: "Error: Failed to update satellite state",
				Code:    http.StatusInternalServerError,
			}
			HandleAppError(w, err)
			return
		}

		configObject, err := fetchSatelliteConfig(r.Context(), q, sat.ID)
		if err != nil {
			log.Printf("Error: Failed to fetch Satellite config: %v", err)
			HandleAppError(w, err)
			return
		}

		// Update state artifact
		err = utils.CreateOrUpdateSatStateArtifact(r.Context(), sat.Name, groupStates, configObject.ConfigName)
		if err != nil {
			log.Println(err)
			err := &AppError{
				Message: "Error: Failed to update satellite state",
				Code:    http.StatusInternalServerError,
			}
			HandleAppError(w, err)
			return
		}
	}

	if err := q.DeleteGroup(r.Context(), group.ID); err != nil {
		log.Println(err)
		err := &AppError{
			Message: "Error: Failed to delete group",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Commit failed: %v", err)
		HandleAppError(w, &AppError{
			Message: "Error: Could not commit transaction",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	committed = true

	err = utils.DeleteArtifact(utils.ConstructHarborDeleteURL(groupName, "group"))
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: "Error: Failed to delete group state",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, map[string]string{})
}

func (s *Server) listGroupHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.dbQueries.ListGroups(r.Context())
	if err != nil {
		log.Printf("Could not list groups: %v", err)
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, result)
}

// groupSatelliteHandler lists all satellites attached to a specific group
func (s *Server) groupSatelliteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupName := vars["group"]

	q := s.dbQueries

	// Get the group by name
	exists, err := q.CheckGroupExists(r.Context(), groupName)
	if err != nil {
		log.Printf("error: failed to get group : %v", err)
		err := &AppError{
			Message: "error: failed to get group",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	if !exists {
		err := &AppError{
			Message: "error: group not found",
			Code:    http.StatusNotFound,
		}
		HandleAppError(w, err)
		return
	}

	// Get all satellites attached to this group
	satellites, err := q.GetSatellitesByGroupName(r.Context(), groupName)
	if err != nil {
		log.Printf("error: failed to get satellites for group: %v", err)
		err := &AppError{
			Message: "error: failed to get satellites for group",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	// Return empty array if no satellites in group
	if len(satellites) == 0 {
		WriteJSONResponse(w, http.StatusOK, []database.Satellite{})
		return
	}

	WriteJSONResponse(w, http.StatusOK, satellites)
}
