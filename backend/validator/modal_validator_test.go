package validator

import (
	"testing"

	"coffee-ranker/entity"
)

// モーダル表示Requestのrank_target_id、trigger、page_pathを検証することを確認。
func TestModalValidatorShow(t *testing.T) {
	v := NewModalValidator()

	got, err := v.Show(ShowModalRequest{RankTargetID: 1, Trigger: entity.ModalTriggerFirstVisit, PagePath: " /beans/1 "})
	assertNoError(t, err)
	if got.RankTargetID != 1 || got.Trigger != entity.ModalTriggerFirstVisit || got.PagePath != "/beans/1" {
		t.Fatalf("unexpected show modal request: %+v", got)
	}

	got, err = v.Show(ShowModalRequest{SourceRankTargetID: 1, Trigger: entity.ModalTriggerScrollEnd, PagePath: "/beans/1"})
	assertNoError(t, err)
	if got.SourceRankTargetID != 1 || got.RankTargetID != 0 {
		t.Fatalf("unexpected backend-selected modal request: %+v", got)
	}

	_, err = v.Show(ShowModalRequest{SourceRankTargetID: 0, Trigger: entity.ModalTrigger("unknown"), PagePath: "/beans/1"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Show(ShowModalRequest{RankTargetID: 1, Trigger: entity.ModalTriggerFirstVisit, PagePath: "javascript:alert(1)"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// クリック/クローズRequestのmodal_display_log_idとpage_pathを検証することを確認。
func TestModalValidatorAction(t *testing.T) {
	v := NewModalValidator()

	got, err := v.Action(ModalActionRequest{ModalDisplayLogID: 10, PagePath: " /modal/click "})
	assertNoError(t, err)
	if got.ModalDisplayLogID != 10 || got.PagePath != "/modal/click" {
		t.Fatalf("unexpected modal action request: %+v", got)
	}

	_, err = v.Action(ModalActionRequest{ModalDisplayLogID: 0, PagePath: "/modal/click"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Action(ModalActionRequest{ModalDisplayLogID: 10, PagePath: "https://example.com"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}
