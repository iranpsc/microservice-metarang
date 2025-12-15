package handler

import (
	"context"
	"metargb/support-service/internal/models"
	"metargb/support-service/internal/service"
	"metargb/support-service/internal/utils"

	pbCommon "metargb/shared/pb/common"
	pb "metargb/shared/pb/support"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NoteHandler struct {
	pb.UnimplementedNoteServiceServer
	noteService service.NoteService
}

func NewNoteHandler(noteService service.NoteService) *NoteHandler {
	return &NoteHandler{
		noteService: noteService,
	}
}

func RegisterNoteHandler(grpcServer *grpc.Server, noteService service.NoteService) {
	handler := NewNoteHandler(noteService)
	pb.RegisterNoteServiceServer(grpcServer, handler)
}

func (h *NoteHandler) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.NoteResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	note, err := h.noteService.CreateNote(ctx, req.UserId, req.Title, req.Content, req.Attachment)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create note: %v", err)
	}

	return convertNoteToProto(note), nil
}

func (h *NoteHandler) GetNotes(ctx context.Context, req *pb.GetNotesRequest) (*pb.NotesResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	notes, err := h.noteService.GetNotes(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get notes: %v", err)
	}

	response := &pb.NotesResponse{
		Notes: make([]*pb.NoteResponse, len(notes)),
	}

	for i, note := range notes {
		response.Notes[i] = convertNoteToProto(note)
	}

	return response, nil
}

func (h *NoteHandler) GetNote(ctx context.Context, req *pb.GetNoteRequest) (*pb.NoteResponse, error) {
	if req.NoteId == 0 {
		return nil, status.Error(codes.InvalidArgument, "note_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	note, err := h.noteService.GetNote(ctx, req.NoteId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get note: %v", err)
	}

	if note == nil {
		return nil, status.Error(codes.NotFound, "note not found")
	}

	return convertNoteToProto(note), nil
}

func (h *NoteHandler) UpdateNote(ctx context.Context, req *pb.UpdateNoteRequest) (*pb.NoteResponse, error) {
	if req.NoteId == 0 {
		return nil, status.Error(codes.InvalidArgument, "note_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	note, err := h.noteService.UpdateNote(ctx, req.NoteId, req.UserId, req.Title, req.Content, req.Attachment)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update note: %v", err)
	}

	return convertNoteToProto(note), nil
}

func (h *NoteHandler) DeleteNote(ctx context.Context, req *pb.DeleteNoteRequest) (*pbCommon.Empty, error) {
	if req.NoteId == 0 {
		return nil, status.Error(codes.InvalidArgument, "note_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := h.noteService.DeleteNote(ctx, req.NoteId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete note: %v", err)
	}

	return &pbCommon.Empty{}, nil
}

// Helper function to convert note model to proto response
func convertNoteToProto(note *models.Note) *pb.NoteResponse {
	return &pb.NoteResponse{
		Id:         note.ID,
		Title:      note.Title,
		Content:    note.Content,
		Attachment: note.Attachment,
		Date:       utils.FormatJalaliDate(note.UpdatedAt),
		Time:       utils.FormatJalaliTime(note.UpdatedAt),
	}
}
