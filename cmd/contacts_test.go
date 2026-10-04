package cmd

import (
	"context"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
)

const contactJSON = `{"data":{"id":42,"label":"work","first_name":"Ada","last_name":"Lovelace","email":"ada@example.com"}}`

func TestListContacts(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":[{"id":42,"first_name":"Ada"}],"pagination":{"current_page":1,"per_page":30,"total_entries":1,"total_pages":1}}`)
	resp, err := listContacts(context.Background(), c, "1010", nil)
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/1010/contacts")
	if len(resp.Data) != 1 || resp.Data[0].ID != 42 {
		t.Errorf("unexpected data %+v", resp.Data)
	}
}

func TestGetContact(t *testing.T) {
	c, calls := recordingClient(t, 200, contactJSON)
	ct, err := getContact(context.Background(), c, "1010", 42)
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/1010/contacts/42")
	if contactDisplayName(ct) != "Ada Lovelace" {
		t.Errorf("unexpected name %q", contactDisplayName(ct))
	}
}

func TestCreateContactFromFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "create"}
	addContactFlags(cmd)
	if missing := missingContactFlags(cmd.Flags()); len(missing) != len(requiredContactFlags) {
		t.Errorf("expected all required flags missing, got %v", missing)
	}
	for k, v := range map[string]string{
		"first-name": "Ada", "last-name": "Lovelace", "email": "ada@example.com", "phone": "+1.4015550100",
		"address1": "1 Main St", "city": "Providence", "state-province": "RI", "postal-code": "02903",
		"country": "US", "organization": "Analytical Engines",
	} {
		_ = cmd.Flags().Set(k, v)
	}
	if missing := missingContactFlags(cmd.Flags()); len(missing) != 0 {
		t.Errorf("unexpected missing flags %v", missing)
	}
	contact, n := contactFromFlags(cmd.Flags())
	if n != 10 {
		t.Errorf("expected 10 fields set, got %d", n)
	}

	c, calls := recordingClient(t, 201, contactJSON)
	if _, err := createContact(context.Background(), c, "1010", contact); err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodPost, "/v2/1010/contacts")
	want := map[string]string{
		"first_name": "Ada", "last_name": "Lovelace", "email": "ada@example.com", "phone": "+1.4015550100",
		"address1": "1 Main St", "city": "Providence", "state_province": "RI", "postal_code": "02903",
		"country": "US", "organization_name": "Analytical Engines",
	}
	for k, v := range want {
		if got.Body[k] != v {
			t.Errorf("body[%s] = %v, want %q", k, got.Body[k], v)
		}
	}
	if _, ok := got.Body["label"]; ok {
		t.Errorf("unset label should not be sent: %+v", got.Body)
	}
}

func TestUpdateContactSendsOnlyChangedFields(t *testing.T) {
	cmd := &cobra.Command{Use: "update"}
	addContactFlags(cmd)
	_ = cmd.Flags().Set("phone", "+1.4015550199")
	contact, n := contactFromFlags(cmd.Flags())
	if n != 1 {
		t.Fatalf("expected 1 field, got %d", n)
	}

	c, calls := recordingClient(t, 200, contactJSON)
	if _, err := updateContact(context.Background(), c, "1010", 42, contact); err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodPatch, "/v2/1010/contacts/42")
	if len(got.Body) != 1 || got.Body["phone"] != "+1.4015550199" {
		t.Errorf("unexpected body %+v", got.Body)
	}
}

func TestDeleteContact(t *testing.T) {
	c, calls := recordingClient(t, 204, "")
	if err := deleteContact(context.Background(), c, "1010", 42); err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodDelete, "/v2/1010/contacts/42")
}

func TestDeleteContactRequiresYes(t *testing.T) {
	err := contactsDeleteCmd.RunE(contactsDeleteCmd, []string{"42"})
	if _, ok := err.(errNeedsYes); !ok {
		t.Fatalf("expected errNeedsYes, got %v", err)
	}
}
