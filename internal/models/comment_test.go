package models

import "testing"

func TestCommentModel(t *testing.T) {
	c := &Comment{
		ChangeID:     1,
		DocumentType: "proposal",
		QuotedText:   "selected text",
		Body:         "Needs clarification",
		UserID:       1,
		Resolved:     false,
	}

	if c.Resolved {
		t.Error("new comment should not be resolved")
	}
}

func TestCommentReplyModel(t *testing.T) {
	r := &CommentReply{
		CommentID: 1,
		UserID:    2,
		Body:      "Fixed it",
	}

	if r.CommentID != 1 {
		t.Errorf("expected comment_id 1, got %d", r.CommentID)
	}
}
