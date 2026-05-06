package pairing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pact/internal/audit"
	"pact/internal/protocol"
	"pact/internal/store"
)

func InitProfile(paths store.Paths, profileName string) (protocol.Profile, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return protocol.Profile{}, fmt.Errorf("profile is required")
	}
	if err := paths.EnsureProfileDirs(profileName); err != nil {
		return protocol.Profile{}, err
	}

	humanID, err := protocol.NewID("human")
	if err != nil {
		return protocol.Profile{}, err
	}
	gatewayID, err := protocol.NewID("gateway")
	if err != nil {
		return protocol.Profile{}, err
	}

	now := time.Now().UTC()
	profile := protocol.Profile{
		Profile:     profileName,
		HumanID:     humanID,
		GatewayID:   gatewayID,
		DisplayName: displayName(profileName),
		CreatedAt:   now,
	}

	if err := store.WriteJSON(paths.ProfileFile(profileName), profile); err != nil {
		return protocol.Profile{}, err
	}
	if err := store.WriteJSONArray[protocol.Contact](paths.ContactsFile(profileName), []protocol.Contact{}); err != nil {
		return protocol.Profile{}, err
	}
	if err := store.WriteJSONArray[protocol.RequestRecord](paths.RequestsFile(profileName), []protocol.RequestRecord{}); err != nil {
		return protocol.Profile{}, err
	}
	if err := store.WriteJSONArray[protocol.GrantRecord](paths.GrantsFile(profileName), []protocol.GrantRecord{}); err != nil {
		return protocol.Profile{}, err
	}

	if err := appendAudit(paths, profile, "profile.initialized", profileName, "", fmt.Sprintf("Initialized profile %s.", profileName)); err != nil {
		return protocol.Profile{}, err
	}
	return profile, nil
}

func LoadProfile(paths store.Paths, profileName string) (protocol.Profile, error) {
	var profile protocol.Profile
	if err := store.ReadJSON(paths.ProfileFile(profileName), &profile); err != nil {
		return protocol.Profile{}, fmt.Errorf("load profile %s: %w", profileName, err)
	}
	return profile, nil
}

func CreateInvite(paths store.Paths, fromProfile string) (string, protocol.Invite, error) {
	profile, err := LoadProfile(paths, fromProfile)
	if err != nil {
		return "", protocol.Invite{}, err
	}

	id, err := protocol.NewID("pinv")
	if err != nil {
		return "", protocol.Invite{}, err
	}
	invite := protocol.Invite{
		ID:        id,
		Type:      protocol.MessagePairInvite,
		Version:   protocol.Version,
		CreatedAt: time.Now().UTC(),
		Human: protocol.InviteHuman{
			ID:          profile.HumanID,
			DisplayName: profile.DisplayName,
		},
		Gateway: protocol.InviteGateway{
			ID: profile.GatewayID,
			Delivery: protocol.Delivery{
				Type:    "local",
				Address: paths.InboxDir(fromProfile),
			},
		},
		SuggestedHandle: fromProfile,
	}

	invitePath := filepath.Join(paths.InvitesDir(), id+".json")
	if err := store.WriteJSON(invitePath, invite); err != nil {
		return "", protocol.Invite{}, err
	}
	if err := appendAudit(paths, profile, "pair.invite_created", id, protocol.MessagePairInvite, fmt.Sprintf("Created invite %s.", id)); err != nil {
		return "", protocol.Invite{}, err
	}
	return invitePath, invite, nil
}

func AcceptInvite(paths store.Paths, asProfile string, handle string, invitePath string) (protocol.Contact, error) {
	profile, err := LoadProfile(paths, asProfile)
	if err != nil {
		return protocol.Contact{}, err
	}

	var invite protocol.Invite
	if err := store.ReadJSON(invitePath, &invite); err != nil {
		return protocol.Contact{}, fmt.Errorf("read invite: %w", err)
	}
	if invite.Type != protocol.MessagePairInvite {
		return protocol.Contact{}, fmt.Errorf("not a pair invite: %s", invite.Type)
	}
	handle = strings.TrimSpace(handle)
	if handle == "" {
		handle = invite.SuggestedHandle
	}
	if handle == "" {
		return protocol.Contact{}, fmt.Errorf("handle is required")
	}

	contact := protocol.Contact{
		Handle:      handle,
		HumanID:     invite.Human.ID,
		DisplayName: invite.Human.DisplayName,
		GatewayID:   invite.Gateway.ID,
		Delivery:    invite.Gateway.Delivery,
		CreatedAt:   time.Now().UTC(),
	}

	contacts, err := store.ReadJSONArray[protocol.Contact](paths.ContactsFile(asProfile))
	if err != nil {
		return protocol.Contact{}, err
	}
	replaced := false
	for i := range contacts {
		if contacts[i].Handle == handle {
			contacts[i] = contact
			replaced = true
			break
		}
	}
	if !replaced {
		contacts = append(contacts, contact)
	}
	if err := store.WriteJSONArray(paths.ContactsFile(asProfile), contacts); err != nil {
		return protocol.Contact{}, err
	}

	msgID, err := protocol.NewID("msg")
	if err != nil {
		return protocol.Contact{}, err
	}
	body, err := json.Marshal(contact)
	if err != nil {
		return protocol.Contact{}, err
	}
	envelope := protocol.Envelope{
		ID:        msgID,
		Type:      protocol.MessagePairAccepted,
		Version:   protocol.Version,
		CreatedAt: time.Now().UTC(),
		From: protocol.PartyRef{
			Profile:   profile.Profile,
			HumanID:   profile.HumanID,
			GatewayID: profile.GatewayID,
		},
		To: protocol.PartyRef{
			Profile:   handle,
			HumanID:   invite.Human.ID,
			GatewayID: invite.Gateway.ID,
		},
		CorrelationID: invite.ID,
		Body:          body,
	}
	if err := store.WriteJSON(filepath.Join(paths.OutboxDir(asProfile), msgID+".json"), envelope); err != nil {
		return protocol.Contact{}, err
	}

	if err := appendAudit(paths, profile, "pair.accepted", invite.ID, protocol.MessagePairAccepted, fmt.Sprintf("Paired %s with %s.", asProfile, handle)); err != nil {
		return protocol.Contact{}, err
	}
	return contact, nil
}

func appendAudit(paths store.Paths, profile protocol.Profile, eventType string, subjectID string, messageType string, summary string) error {
	id, err := protocol.NewID("aud")
	if err != nil {
		return err
	}
	return audit.Append(paths.AuditFile(profile.Profile), protocol.AuditEvent{
		ID:             id,
		Timestamp:      time.Now().UTC(),
		Profile:        profile.Profile,
		ActorGatewayID: profile.GatewayID,
		EventType:      eventType,
		SubjectID:      subjectID,
		MessageType:    messageType,
		Summary:        summary,
	})
}

func displayName(profileName string) string {
	if profileName == "" {
		return ""
	}
	return strings.ToUpper(profileName[:1]) + profileName[1:]
}

func InvitePathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
