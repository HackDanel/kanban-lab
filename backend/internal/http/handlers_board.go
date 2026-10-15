package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

func dateOf(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	return &d.Time
}

// --- boards ---

func (h *handlers) ListBoards(ctx context.Context, req api.ListBoardsRequestObject) (api.ListBoardsResponseObject, error) {
	boards, err := h.svc.ListBoards(ctx, actorFrom(ctx), req.ProjectId)
	if err != nil {
		return nil, err
	}
	return api.ListBoards200JSONResponse(mapSlice(boards, toBoard)), nil
}

func (h *handlers) CreateBoard(ctx context.Context, req api.CreateBoardRequestObject) (api.CreateBoardResponseObject, error) {
	board, err := h.svc.CreateBoard(ctx, actorFrom(ctx), req.ProjectId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return api.CreateBoard201JSONResponse(toBoardDetail(board)), nil
}

func (h *handlers) GetBoard(ctx context.Context, req api.GetBoardRequestObject) (api.GetBoardResponseObject, error) {
	board, err := h.svc.GetBoard(ctx, actorFrom(ctx), req.BoardId)
	if err != nil {
		return nil, err
	}
	return api.GetBoard200JSONResponse(toBoardDetail(board)), nil
}

func (h *handlers) UpdateBoard(ctx context.Context, req api.UpdateBoardRequestObject) (api.UpdateBoardResponseObject, error) {
	board, err := h.svc.RenameBoard(ctx, actorFrom(ctx), req.BoardId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return api.UpdateBoard200JSONResponse(toBoard(board)), nil
}

func (h *handlers) DeleteBoard(ctx context.Context, req api.DeleteBoardRequestObject) (api.DeleteBoardResponseObject, error) {
	if err := h.svc.DeleteBoard(ctx, actorFrom(ctx), req.BoardId); err != nil {
		return nil, err
	}
	return api.DeleteBoard204Response{}, nil
}

// --- columns ---

func (h *handlers) CreateColumn(ctx context.Context, req api.CreateColumnRequestObject) (api.CreateColumnResponseObject, error) {
	limit, _ := optional(req.Body.WipLimit)
	column, err := h.svc.CreateColumn(ctx, actorFrom(ctx), req.BoardId, req.Body.Name, limit)
	if err != nil {
		return nil, err
	}
	return api.CreateColumn201JSONResponse(toColumn(column)), nil
}

func (h *handlers) UpdateColumn(ctx context.Context, req api.UpdateColumnRequestObject) (api.UpdateColumnResponseObject, error) {
	limit, setLimit := optional(req.Body.WipLimit)
	column, err := h.svc.UpdateColumn(ctx, actorFrom(ctx), req.ColumnId, service.ColumnPatch{
		Name: req.Body.Name, SetWipLimit: setLimit, WipLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	return api.UpdateColumn200JSONResponse(toColumn(column)), nil
}

func (h *handlers) DeleteColumn(ctx context.Context, req api.DeleteColumnRequestObject) (api.DeleteColumnResponseObject, error) {
	if err := h.svc.DeleteColumn(ctx, actorFrom(ctx), req.ColumnId); err != nil {
		return nil, err
	}
	return api.DeleteColumn204Response{}, nil
}

func (h *handlers) MoveColumn(ctx context.Context, req api.MoveColumnRequestObject) (api.MoveColumnResponseObject, error) {
	after, _ := optional(req.Body.AfterColumnId)
	column, err := h.svc.MoveColumn(ctx, actorFrom(ctx), req.ColumnId, after)
	if err != nil {
		return nil, err
	}
	return api.MoveColumn200JSONResponse(toColumn(column)), nil
}

// --- cards ---

func (h *handlers) ListCards(ctx context.Context, req api.ListCardsRequestObject) (api.ListCardsResponseObject, error) {
	p := req.Params
	filter := service.CardFilter{
		Query: deref(p.Q), ColumnID: p.ColumnId, AssigneeID: p.AssigneeId,
		Page: pageOf(p.Limit, p.Offset),
	}
	if p.LabelId != nil {
		filter.LabelIDs = *p.LabelId
	}
	if p.Priority != nil {
		priority := string(*p.Priority)
		filter.Priority = &priority
	}
	cards, total, err := h.svc.ListCards(ctx, actorFrom(ctx), req.BoardId, filter)
	if err != nil {
		return nil, err
	}
	return api.ListCards200JSONResponse{Items: mapSlice(cards, toCard), Total: int(total)}, nil
}

func (h *handlers) CreateCard(ctx context.Context, req api.CreateCardRequestObject) (api.CreateCardResponseObject, error) {
	b := req.Body
	assignee, _ := optional(b.AssigneeId)
	due, _ := optional(b.DueDate)
	in := service.CardInput{
		Title: b.Title, Description: deref(b.Description), AssigneeID: assignee, DueDate: dateOf(due),
	}
	if b.Priority != nil {
		in.Priority = string(*b.Priority)
	}
	if b.LabelIds != nil {
		in.LabelIDs = *b.LabelIds
	}
	card, err := h.svc.CreateCard(ctx, actorFrom(ctx), req.ColumnId, in)
	if err != nil {
		return nil, err
	}
	return api.CreateCard201JSONResponse(toCard(card)), nil
}

func (h *handlers) GetCard(ctx context.Context, req api.GetCardRequestObject) (api.GetCardResponseObject, error) {
	card, err := h.svc.GetCard(ctx, actorFrom(ctx), req.CardId)
	if err != nil {
		return nil, err
	}
	return api.GetCard200JSONResponse(toCard(card)), nil
}

func (h *handlers) UpdateCard(ctx context.Context, req api.UpdateCardRequestObject) (api.UpdateCardResponseObject, error) {
	b := req.Body
	assignee, setAssignee := optional(b.AssigneeId)
	due, setDue := optional(b.DueDate)
	patch := service.CardPatch{
		Title: b.Title, Description: b.Description, LabelIDs: b.LabelIds,
		SetAssignee: setAssignee, AssigneeID: assignee, SetDueDate: setDue, DueDate: dateOf(due),
	}
	if b.Priority != nil {
		priority := string(*b.Priority)
		patch.Priority = &priority
	}
	card, err := h.svc.UpdateCard(ctx, actorFrom(ctx), req.CardId, patch)
	if err != nil {
		return nil, err
	}
	return api.UpdateCard200JSONResponse(toCard(card)), nil
}

func (h *handlers) DeleteCard(ctx context.Context, req api.DeleteCardRequestObject) (api.DeleteCardResponseObject, error) {
	if err := h.svc.DeleteCard(ctx, actorFrom(ctx), req.CardId); err != nil {
		return nil, err
	}
	return api.DeleteCard204Response{}, nil
}

func (h *handlers) MoveCard(ctx context.Context, req api.MoveCardRequestObject) (api.MoveCardResponseObject, error) {
	after, _ := optional(req.Body.AfterCardId)
	card, err := h.svc.MoveCard(ctx, actorFrom(ctx), req.CardId, req.Body.ColumnId, after)
	if err != nil {
		return nil, err
	}
	return api.MoveCard200JSONResponse(toCard(card)), nil
}

func (h *handlers) ListCardActivity(ctx context.Context, req api.ListCardActivityRequestObject) (api.ListCardActivityResponseObject, error) {
	entries, err := h.svc.ListActivity(ctx, actorFrom(ctx), req.CardId, pageOf(req.Params.Limit, req.Params.Offset))
	if err != nil {
		return nil, err
	}
	return api.ListCardActivity200JSONResponse(mapSlice(entries, toActivity)), nil
}

// --- comments ---

func (h *handlers) ListComments(ctx context.Context, req api.ListCommentsRequestObject) (api.ListCommentsResponseObject, error) {
	comments, err := h.svc.ListComments(ctx, actorFrom(ctx), req.CardId)
	if err != nil {
		return nil, err
	}
	return api.ListComments200JSONResponse(mapSlice(comments, toComment)), nil
}

func (h *handlers) CreateComment(ctx context.Context, req api.CreateCommentRequestObject) (api.CreateCommentResponseObject, error) {
	comment, err := h.svc.CreateComment(ctx, actorFrom(ctx), req.CardId, req.Body.Body)
	if err != nil {
		return nil, err
	}
	return api.CreateComment201JSONResponse(toComment(comment)), nil
}

func (h *handlers) UpdateComment(ctx context.Context, req api.UpdateCommentRequestObject) (api.UpdateCommentResponseObject, error) {
	comment, err := h.svc.UpdateComment(ctx, actorFrom(ctx), req.CommentId, req.Body.Body)
	if err != nil {
		return nil, err
	}
	return api.UpdateComment200JSONResponse(toComment(comment)), nil
}

func (h *handlers) DeleteComment(ctx context.Context, req api.DeleteCommentRequestObject) (api.DeleteCommentResponseObject, error) {
	if err := h.svc.DeleteComment(ctx, actorFrom(ctx), req.CommentId); err != nil {
		return nil, err
	}
	return api.DeleteComment204Response{}, nil
}

// --- attachments ---

func (h *handlers) ListAttachments(ctx context.Context, req api.ListAttachmentsRequestObject) (api.ListAttachmentsResponseObject, error) {
	attachments, err := h.svc.ListAttachments(ctx, actorFrom(ctx), req.CardId)
	if err != nil {
		return nil, err
	}
	return api.ListAttachments200JSONResponse(mapSlice(attachments, toAttachment)), nil
}

var errNoFile = domain.Invalid("file", "Выберите файл")

func (h *handlers) UploadAttachment(ctx context.Context, req api.UploadAttachmentRequestObject) (api.UploadAttachmentResponseObject, error) {
	if req.Body == nil {
		return nil, errNoFile
	}
	// The upload is streamed straight to storage instead of being buffered in memory.
	for {
		part, err := req.Body.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, errNoFile
		}
		if err != nil {
			return nil, domain.Invalid("file", "Некорректный multipart-запрос")
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		attachment, err := h.svc.UploadAttachment(ctx, actorFrom(ctx), req.CardId,
			part.FileName(), part.Header.Get("Content-Type"), part)
		_ = part.Close()
		if err != nil {
			return nil, err
		}
		return api.UploadAttachment201JSONResponse(toAttachment(attachment)), nil
	}
}

func (h *handlers) DeleteAttachment(ctx context.Context, req api.DeleteAttachmentRequestObject) (api.DeleteAttachmentResponseObject, error) {
	if err := h.svc.DeleteAttachment(ctx, actorFrom(ctx), req.AttachmentId); err != nil {
		return nil, err
	}
	return api.DeleteAttachment204Response{}, nil
}

// downloadResponse streams an attachment with headers that make the browser
// save it rather than render it.
type downloadResponse struct {
	download service.Download
}

func (d downloadResponse) VisitDownloadAttachmentResponse(w http.ResponseWriter) error {
	defer func() { _ = d.download.Content.Close() }()
	a := d.download.Attachment
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.Filename}))
	w.Header().Set("Content-Length", fmt.Sprint(a.Size))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, err := io.Copy(w, d.download.Content)
	return err
}

func (h *handlers) DownloadAttachment(ctx context.Context, req api.DownloadAttachmentRequestObject) (api.DownloadAttachmentResponseObject, error) {
	download, err := h.svc.OpenAttachment(ctx, actorFrom(ctx), req.AttachmentId)
	if err != nil {
		return nil, err
	}
	return downloadResponse{download: download}, nil
}
