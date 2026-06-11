package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ArtemSafin/Domely/services/task-service/internal/model"
	"github.com/ArtemSafin/Domely/services/task-service/internal/service"
)

type Handler struct {
	svc *service.TaskService
}

func New(svc *service.TaskService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.health)
	r.Post("/users", h.createUser)
	r.Get("/users/telegram/{telegramID}", h.getUserByTelegramID)
	r.Post("/houses", h.createHouse)
	r.Get("/houses/user/{userID}", h.getHousesByUser)
	r.Post("/houses/{houseID}/members", h.inviteMember)
	r.Get("/houses/{houseID}/members", h.getHouseMembers)
	r.Post("/tasks", h.createTask)
	r.Get("/tasks/house/{houseID}", h.getTasksByHouse)
	r.Get("/tasks/assigned/{userID}", h.getTasksByAssignedTo)
	r.Get("/tasks/{taskID}", h.getTaskByID)
	r.Put("/tasks/{taskID}", h.updateTask)
	r.Delete("/tasks/{taskID}", h.deleteTask)
	r.Post("/tasks/{taskID}/complete", h.completeTask)
	r.Get("/tasks/history/{houseID}", h.getTaskHistory)
	r.Get("/reminders/pending", h.getPendingReminders)
	r.Post("/reminders/{reminderID}/sent", h.markReminderSent)
	return r
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TelegramID int64  `json:"telegram_id"`
		Name       string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.svc.RegisterUser(r.Context(), req.TelegramID, req.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, user)
}

func (h *Handler) getUserByTelegramID(w http.ResponseWriter, r *http.Request) {
	telegramID := int64(0)
	if _, err := fmt.Sscan(chi.URLParam(r, "telegramID"), &telegramID); err != nil {
		respondError(w, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	user, err := h.svc.GetUserByTelegramID(r.Context(), telegramID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	respond(w, http.StatusOK, user)
}

func (h *Handler) createHouse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string    `json:"name"`
		OwnerID uuid.UUID `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	house, err := h.svc.CreateHouse(r.Context(), req.Name, req.OwnerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, house)
}

func (h *Handler) getHousesByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	houses, err := h.svc.GetHousesByUserID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, houses)
}

func (h *Handler) inviteMember(w http.ResponseWriter, r *http.Request) {
	houseID, err := uuid.Parse(chi.URLParam(r, "houseID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid house_id")
		return
	}
	var req struct {
		UserID uuid.UUID `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.InviteMember(r.Context(), houseID, req.UserID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) getHouseMembers(w http.ResponseWriter, r *http.Request) {
	houseID, err := uuid.Parse(chi.URLParam(r, "houseID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid house_id")
		return
	}
	members, err := h.svc.GetHouseMembers(r.Context(), houseID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, members)
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	task, err := h.svc.CreateTask(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, task)
}

func (h *Handler) getTasksByHouse(w http.ResponseWriter, r *http.Request) {
	houseID, err := uuid.Parse(chi.URLParam(r, "houseID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid house_id")
		return
	}
	tasks, err := h.svc.GetTasksByHouseID(r.Context(), houseID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, tasks)
}

func (h *Handler) getTasksByAssignedTo(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	tasks, err := h.svc.GetTasksByAssignedTo(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, tasks)
}

func (h *Handler) getTaskByID(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task_id")
		return
	}
	task, err := h.svc.GetTasksByHouseID(r.Context(), taskID)
	if err != nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}
	respond(w, http.StatusOK, task)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task_id")
		return
	}
	var req model.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	task, err := h.svc.UpdateTask(r.Context(), taskID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, task)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task_id")
		return
	}
	if err := h.svc.DeleteTask(r.Context(), taskID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) completeTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task_id")
		return
	}
	var req model.CompleteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.CompleteTask(r.Context(), taskID, req.CompletedBy); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) getTaskHistory(w http.ResponseWriter, r *http.Request) {
	houseID, err := uuid.Parse(chi.URLParam(r, "houseID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid house_id")
		return
	}
	history, err := h.svc.GetTaskHistory(r.Context(), houseID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, history)
}

func (h *Handler) getPendingReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := h.svc.GetPendingReminders(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, reminders)
}

func (h *Handler) markReminderSent(w http.ResponseWriter, r *http.Request) {
	reminderID, err := uuid.Parse(chi.URLParam(r, "reminderID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid reminder_id")
		return
	}
	if err := h.svc.MarkReminderSent(r.Context(), reminderID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
