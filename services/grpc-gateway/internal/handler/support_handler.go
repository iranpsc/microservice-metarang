package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbAuth "metargb/shared/pb/auth"
	pbCommon "metargb/shared/pb/common"
	pbSupport "metargb/shared/pb/support"
)

type SupportHandler struct {
	ticketClient    pbSupport.TicketServiceClient
	reportClient    pbSupport.ReportServiceClient
	userEventClient pbSupport.UserEventReportServiceClient
	noteClient      pbSupport.NoteServiceClient
	authClient      pbAuth.AuthServiceClient
}

func NewSupportHandler(supportConn, authConn *grpc.ClientConn) *SupportHandler {
	return &SupportHandler{
		ticketClient:    pbSupport.NewTicketServiceClient(supportConn),
		reportClient:    pbSupport.NewReportServiceClient(supportConn),
		userEventClient: pbSupport.NewUserEventReportServiceClient(supportConn),
		noteClient:      pbSupport.NewNoteServiceClient(supportConn),
		authClient:      pbAuth.NewAuthServiceClient(authConn),
	}
}

// Helper function to get authenticated user ID
func (h *SupportHandler) getAuthUserID(r *http.Request) (uint64, error) {
	token := extractTokenFromHeader(r)
	if token == "" {
		return 0, status.Error(codes.Unauthenticated, "authentication required")
	}

	validateReq := &pbAuth.ValidateTokenRequest{Token: token}
	validateResp, err := h.authClient.ValidateToken(r.Context(), validateReq)
	if err != nil || !validateResp.Valid {
		return 0, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	return validateResp.UserId, nil
}

// ============================================================================
// Tickets API
// ============================================================================

// ListTickets handles GET /api/tickets
// Query params: page, cursor, recieved (bool)
func (h *SupportHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Parse pagination
	page := int32(1)
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.ParseInt(p, 10, 32); err == nil {
			page = int32(parsed)
		}
	}

	perPage := int32(10)
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if parsed, err := strconv.ParseInt(pp, 10, 32); err == nil {
			perPage = int32(parsed)
		}
	}

	// Parse received parameter (not used but kept for future use)
	_ = r.URL.Query().Get("recieved")

	grpcReq := &pbSupport.GetTicketsRequest{
		UserId: userID,
		Pagination: &pbCommon.PaginationRequest{
			Page:    page,
			PerPage: perPage,
		},
	}

	resp, err := h.ticketClient.GetTickets(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert to Laravel-compatible format
	tickets := make([]map[string]interface{}, 0, len(resp.Tickets))
	for _, ticket := range resp.Tickets {
		ticketMap := map[string]interface{}{
			"id":      ticket.Id,
			"title":   ticket.Title,
			"content": ticket.Content,
			"code":    ticket.Code,
			"status":  ticket.Status,
			"date":    ticket.UpdatedAt,
			"time":    ticket.UpdatedAt,
		}

		if ticket.Sender != nil {
			ticketMap["sender"] = map[string]interface{}{
				"name":          ticket.Sender.Name,
				"code":          ticket.Sender.Code,
				"profile-photo": ticket.Sender.ProfilePhoto,
			}
		}

		if ticket.Receiver != nil {
			ticketMap["reciever"] = map[string]interface{}{
				"name":          ticket.Receiver.Name,
				"code":          ticket.Receiver.Code,
				"profile-photo": ticket.Receiver.ProfilePhoto,
			}
		}

		if ticket.Department != "" {
			ticketMap["department"] = ticket.Department
		}

		if ticket.Attachment != "" {
			ticketMap["attachment"] = ticket.Attachment
		}

		if len(ticket.Responses) > 0 {
			responses := make([]map[string]interface{}, 0, len(ticket.Responses))
			for _, resp := range ticket.Responses {
				responses = append(responses, map[string]interface{}{
					"id":             resp.Id,
					"response":       resp.Response,
					"attachment":     resp.Attachment,
					"responser_name": resp.ResponserName,
					"responser_id":   resp.ResponserId,
					"created_at":     resp.CreatedAt,
				})
			}
			ticketMap["responses"] = responses
		}

		tickets = append(tickets, ticketMap)
	}

	// Simple pagination response (matching Laravel simplePaginate)
	response := map[string]interface{}{
		"data": tickets,
	}
	if len(tickets) == int(perPage) {
		response["next_page_url"] = r.URL.Path + "?page=" + strconv.Itoa(int(page+1))
	}

	writeJSON(w, http.StatusOK, response)
}

// CreateTicket handles POST /api/tickets
func (h *SupportHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	var req struct {
		Title      string  `json:"title"`
		Content    string  `json:"content"`
		Attachment string  `json:"attachment"`
		Reciever   *uint64 `json:"reciever"` // Note: Laravel uses 'reciever' (typo)
		Department string  `json:"department"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pbSupport.CreateTicketRequest{
		UserId:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Attachment: req.Attachment,
	}

	if req.Reciever != nil {
		grpcReq.ReceiverId = *req.Reciever
	}
	if req.Department != "" {
		grpcReq.Department = req.Department
	}

	resp, err := h.ticketClient.CreateTicket(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert to Laravel-compatible format
	ticketMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"code":    resp.Code,
		"status":  resp.Status,
		"date":    resp.CreatedAt,
		"time":    resp.CreatedAt,
	}

	if resp.Sender != nil {
		ticketMap["sender"] = map[string]interface{}{
			"name":          resp.Sender.Name,
			"code":          resp.Sender.Code,
			"profile-photo": resp.Sender.ProfilePhoto,
		}
	}

	if resp.Receiver != nil {
		ticketMap["reciever"] = map[string]interface{}{
			"name":          resp.Receiver.Name,
			"code":          resp.Receiver.Code,
			"profile-photo": resp.Receiver.ProfilePhoto,
		}
	}

	if resp.Department != "" {
		ticketMap["department"] = resp.Department
	}

	writeJSON(w, http.StatusCreated, ticketMap)
}

// GetTicket handles GET /api/tickets/{ticket}
func (h *SupportHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	ticketIDStr := extractIDFromPath(r.URL.Path, "/api/tickets/")
	if ticketIDStr == "" {
		writeError(w, http.StatusBadRequest, "ticket_id is required")
		return
	}

	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket_id")
		return
	}

	grpcReq := &pbSupport.GetTicketRequest{
		TicketId: ticketID,
		UserId:   userID,
	}

	resp, err := h.ticketClient.GetTicket(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert to Laravel-compatible format (same as CreateTicket)
	ticketMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"code":    resp.Code,
		"status":  resp.Status,
		"date":    resp.UpdatedAt,
		"time":    resp.UpdatedAt,
	}

	if resp.Sender != nil {
		ticketMap["sender"] = map[string]interface{}{
			"name":          resp.Sender.Name,
			"code":          resp.Sender.Code,
			"profile-photo": resp.Sender.ProfilePhoto,
		}
	}

	if resp.Receiver != nil {
		ticketMap["reciever"] = map[string]interface{}{
			"name":          resp.Receiver.Name,
			"code":          resp.Receiver.Code,
			"profile-photo": resp.Receiver.ProfilePhoto,
		}
	}

	if resp.Department != "" {
		ticketMap["department"] = resp.Department
	}

	if len(resp.Responses) > 0 {
		responses := make([]map[string]interface{}, 0, len(resp.Responses))
		for _, resp := range resp.Responses {
			responses = append(responses, map[string]interface{}{
				"id":             resp.Id,
				"response":       resp.Response,
				"attachment":     resp.Attachment,
				"responser_name": resp.ResponserName,
				"responser_id":   resp.ResponserId,
				"created_at":     resp.CreatedAt,
			})
		}
		ticketMap["responses"] = responses
	}

	writeJSON(w, http.StatusOK, ticketMap)
}

// UpdateTicket handles PUT/PATCH /api/tickets/{ticket}
func (h *SupportHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	ticketIDStr := extractIDFromPath(r.URL.Path, "/api/tickets/")
	if ticketIDStr == "" {
		writeError(w, http.StatusBadRequest, "ticket_id is required")
		return
	}

	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket_id")
		return
	}

	var req struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		Attachment string `json:"attachment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pbSupport.UpdateTicketRequest{
		TicketId:   ticketID,
		UserId:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Attachment: req.Attachment,
	}

	resp, err := h.ticketClient.UpdateTicket(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert response (same format as GetTicket)
	ticketMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"code":    resp.Code,
		"status":  resp.Status,
		"date":    resp.UpdatedAt,
		"time":    resp.UpdatedAt,
	}

	if resp.Sender != nil {
		ticketMap["sender"] = map[string]interface{}{
			"name":          resp.Sender.Name,
			"code":          resp.Sender.Code,
			"profile-photo": resp.Sender.ProfilePhoto,
		}
	}

	writeJSON(w, http.StatusOK, ticketMap)
}

// AddTicketResponse handles POST /api/tickets/response/{ticket}
func (h *SupportHandler) AddTicketResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	ticketIDStr := extractIDFromPath(r.URL.Path, "/api/tickets/response/")
	if ticketIDStr == "" {
		writeError(w, http.StatusBadRequest, "ticket_id is required")
		return
	}

	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket_id")
		return
	}

	var req struct {
		Response   string `json:"response"`
		Attachment string `json:"attachment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pbSupport.AddResponseRequest{
		TicketId:   ticketID,
		UserId:     userID,
		Response:   req.Response,
		Attachment: req.Attachment,
	}

	resp, err := h.ticketClient.AddResponse(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Return updated ticket (same format as GetTicket)
	ticketMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"code":    resp.Code,
		"status":  resp.Status,
		"date":    resp.UpdatedAt,
		"time":    resp.UpdatedAt,
	}

	if len(resp.Responses) > 0 {
		responses := make([]map[string]interface{}, 0, len(resp.Responses))
		for _, resp := range resp.Responses {
			responses = append(responses, map[string]interface{}{
				"id":             resp.Id,
				"response":       resp.Response,
				"attachment":     resp.Attachment,
				"responser_name": resp.ResponserName,
				"responser_id":   resp.ResponserId,
				"created_at":     resp.CreatedAt,
			})
		}
		ticketMap["responses"] = responses
	}

	writeJSON(w, http.StatusOK, ticketMap)
}

// CloseTicket handles GET /api/tickets/close/{ticket}
func (h *SupportHandler) CloseTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	ticketIDStr := extractIDFromPath(r.URL.Path, "/api/tickets/close/")
	if ticketIDStr == "" {
		writeError(w, http.StatusBadRequest, "ticket_id is required")
		return
	}

	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket_id")
		return
	}

	grpcReq := &pbSupport.CloseTicketRequest{
		TicketId: ticketID,
		UserId:   userID,
	}

	resp, err := h.ticketClient.CloseTicket(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Return updated ticket
	ticketMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"code":    resp.Code,
		"status":  resp.Status,
		"date":    resp.UpdatedAt,
		"time":    resp.UpdatedAt,
	}

	writeJSON(w, http.StatusOK, ticketMap)
}

// ============================================================================
// Reports API
// ============================================================================

// ListReports handles GET /api/reports
func (h *SupportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	page := int32(1)
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.ParseInt(p, 10, 32); err == nil {
			page = int32(parsed)
		}
	}

	perPage := int32(10)
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if parsed, err := strconv.ParseInt(pp, 10, 32); err == nil {
			perPage = int32(parsed)
		}
	}

	grpcReq := &pbSupport.GetReportsRequest{
		UserId: userID,
		Pagination: &pbCommon.PaginationRequest{
			Page:    page,
			PerPage: perPage,
		},
	}

	resp, err := h.reportClient.GetReports(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	reports := make([]map[string]interface{}, 0, len(resp.Reports))
	for _, report := range resp.Reports {
		reportMap := map[string]interface{}{
			"id":       report.Id,
			"title":    report.Reason,         // Mapping reason to title
			"subject":  report.ReportableType, // Mapping reportable_type to subject
			"content":  report.Description,
			"datetime": report.CreatedAt,
		}
		reports = append(reports, reportMap)
	}

	response := map[string]interface{}{
		"data": reports,
	}
	if len(reports) == int(perPage) {
		response["next_page_url"] = r.URL.Path + "?page=" + strconv.Itoa(int(page+1))
	}

	writeJSON(w, http.StatusOK, response)
}

// CreateReport handles POST /api/reports
func (h *SupportHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	var req struct {
		Subject     string   `json:"subject"`
		Title       string   `json:"title"`
		Content     string   `json:"content"`
		URL         string   `json:"url"`
		Attachments []string `json:"attachments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Map Laravel fields to proto fields
	grpcReq := &pbSupport.CreateReportRequest{
		UserId:         userID,
		ReportableType: req.Subject,
		Reason:         req.Title,
		Description:    req.Content,
	}

	resp, err := h.reportClient.CreateReport(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	reportMap := map[string]interface{}{
		"id":       resp.Id,
		"title":    resp.Reason,
		"subject":  resp.ReportableType,
		"content":  resp.Description,
		"datetime": resp.CreatedAt,
	}

	if req.URL != "" {
		reportMap["url"] = req.URL
	}

	writeJSON(w, http.StatusCreated, reportMap)
}

// GetReport handles GET /api/reports/{report}
func (h *SupportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	reportIDStr := extractIDFromPath(r.URL.Path, "/api/reports/")
	if reportIDStr == "" {
		writeError(w, http.StatusBadRequest, "report_id is required")
		return
	}

	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report_id")
		return
	}

	grpcReq := &pbSupport.GetReportRequest{
		ReportId: reportID,
	}

	resp, err := h.reportClient.GetReport(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	reportMap := map[string]interface{}{
		"id":       resp.Id,
		"title":    resp.Reason,
		"subject":  resp.ReportableType,
		"content":  resp.Description,
		"datetime": resp.CreatedAt,
	}

	writeJSON(w, http.StatusOK, reportMap)
}

// ============================================================================
// Notes API
// ============================================================================

// ListNotes handles GET /api/notes
func (h *SupportHandler) ListNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	grpcReq := &pbSupport.GetNotesRequest{
		UserId: userID,
	}

	resp, err := h.noteClient.GetNotes(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	notes := make([]map[string]interface{}, 0, len(resp.Notes))
	for _, note := range resp.Notes {
		noteMap := map[string]interface{}{
			"id":      note.Id,
			"title":   note.Title,
			"content": note.Content,
			"date":    note.Date,
			"time":    note.Time,
		}
		if note.Attachment != "" {
			noteMap["attachment"] = note.Attachment
		}
		notes = append(notes, noteMap)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": notes})
}

// CreateNote handles POST /api/notes
func (h *SupportHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	var req struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		Attachment string `json:"attachment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pbSupport.CreateNoteRequest{
		UserId:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Attachment: req.Attachment,
	}

	resp, err := h.noteClient.CreateNote(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	noteMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"date":    resp.Date,
		"time":    resp.Time,
	}
	if resp.Attachment != "" {
		noteMap["attachment"] = resp.Attachment
	}

	writeJSON(w, http.StatusCreated, noteMap)
}

// GetNote handles GET /api/notes/{note}
func (h *SupportHandler) GetNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	noteIDStr := extractIDFromPath(r.URL.Path, "/api/notes/")
	if noteIDStr == "" {
		writeError(w, http.StatusBadRequest, "note_id is required")
		return
	}

	noteID, err := strconv.ParseUint(noteIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid note_id")
		return
	}

	grpcReq := &pbSupport.GetNoteRequest{
		NoteId: noteID,
		UserId: userID,
	}

	resp, err := h.noteClient.GetNote(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	noteMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"date":    resp.Date,
		"time":    resp.Time,
	}
	if resp.Attachment != "" {
		noteMap["attachment"] = resp.Attachment
	}

	writeJSON(w, http.StatusOK, noteMap)
}

// UpdateNote handles PUT/PATCH /api/notes/{note}
func (h *SupportHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	noteIDStr := extractIDFromPath(r.URL.Path, "/api/notes/")
	if noteIDStr == "" {
		writeError(w, http.StatusBadRequest, "note_id is required")
		return
	}

	noteID, err := strconv.ParseUint(noteIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid note_id")
		return
	}

	var req struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		Attachment string `json:"attachment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pbSupport.UpdateNoteRequest{
		NoteId:     noteID,
		UserId:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Attachment: req.Attachment,
	}

	resp, err := h.noteClient.UpdateNote(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	noteMap := map[string]interface{}{
		"id":      resp.Id,
		"title":   resp.Title,
		"content": resp.Content,
		"date":    resp.Date,
		"time":    resp.Time,
	}
	if resp.Attachment != "" {
		noteMap["attachment"] = resp.Attachment
	}

	writeJSON(w, http.StatusOK, noteMap)
}

// DeleteNote handles DELETE /api/notes/{note}
func (h *SupportHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getAuthUserID(r)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	noteIDStr := extractIDFromPath(r.URL.Path, "/api/notes/")
	if noteIDStr == "" {
		writeError(w, http.StatusBadRequest, "note_id is required")
		return
	}

	noteID, err := strconv.ParseUint(noteIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid note_id")
		return
	}

	grpcReq := &pbSupport.DeleteNoteRequest{
		NoteId: noteID,
		UserId: userID,
	}

	_, err = h.noteClient.DeleteNote(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
