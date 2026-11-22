package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type Type string

const (
	TypeRSS     Type = "rss"
	TypeYouTube Type = "youtube"
	TypeBluesky Type = "bluesky"
	TypeManual  Type = "manual"

	// SavedLinksName is the canonical display name of the system manual singleton.
	SavedLinksName = "Saved links"
)

type Source struct {
	Config           map[string]any
	Key              string
	ProfileImageID   string
	ID               uuid.UUID
	Name             string
	Type             Type
	RSSURL           string
	YouTubeChannelID string
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewFetchableSource constructs a Source from a registry-prepared config.
func NewFetchableSource(name string, typ Type, config map[string]any, key string) (*Source, error) {
	if typ == TypeManual || typ == "" {
		return nil, apperr.InvalidInput.New("invalid fetchable source type")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.InvalidInput.New("name is required")
	}
	now := time.Now().UTC()
	s := &Source{ID: uuid.New(), Name: name, Type: typ, Active: true, CreatedAt: now, UpdatedAt: now}
	s.setConfig(config, key)
	return s, nil
}

func (s *Source) setConfig(config map[string]any, key string) {
	s.Config, s.Key = config, key
	// The historical columns are kept in sync until their removal in a future migration.
	switch s.Type {
	case TypeRSS:
		s.RSSURL, _ = config["url"].(string)
	case TypeYouTube:
		s.YouTubeChannelID, _ = config["channelId"].(string)
	}
}

// UpdateConfig atomically changes the name and prepared config of a fetchable Source.
func (s *Source) UpdateConfig(name string, config map[string]any, key string) error {
	if s.Type == TypeBluesky && key != s.Key {
		return apperr.InvalidInput.New("bluesky profile cannot be changed")
	}
	if err := s.Rename(name); err != nil {
		return err
	}
	s.setConfig(config, key)
	return nil
}

// NewRSSSource creates a fetchable RSS Source. Type is fixed at construction.
func NewRSSSource(name, feedURL string) (*Source, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.InvalidInput.New("name is required")
	}
	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return nil, apperr.InvalidInput.New("rss url is required")
	}
	return NewFetchableSource(name, TypeRSS, map[string]any{"url": feedURL}, feedURL)
}

// NewYouTubeSource creates a Source for a canonical channel ID.
func NewYouTubeSource(name, channelID string) (*Source, error) {
	name, channelID = strings.TrimSpace(name), strings.TrimSpace(channelID)
	if name == "" {
		return nil, apperr.InvalidInput.New("name is required")
	}
	channelID = strings.TrimPrefix(channelID, "https://www.youtube.com/channel/")
	if !regexp.MustCompile(`^UC[A-Za-z0-9_-]{22}$`).MatchString(channelID) {
		return nil, apperr.InvalidInput.New("enter a YouTube channel ID starting with UC or a canonical https://www.youtube.com/channel/ URL")
	}
	return NewFetchableSource(name, TypeYouTube, map[string]any{"channelId": channelID}, channelID)
}

// NewManualSource creates a system manual Source. Type is fixed at construction.
// Reader create path must not call this; EnsureSavedLinks / bootstrap use it.
func NewManualSource(name string) (*Source, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.InvalidInput.New("name is required")
	}
	now := time.Now().UTC()
	return &Source{
		ID:     uuid.New(),
		Name:   name,
		Type:   TypeManual,
		Config: map[string]any{}, Key: "singleton",
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Deactivate marks a fetchable Source inactive. Manual Sources cannot be deactivated.
func (s *Source) Deactivate() error {
	if s.Type == TypeManual {
		return ErrManualSourceImmutable.New("manual sources cannot be deactivated")
	}
	if !s.Active {
		return nil
	}
	s.Active = false
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// Reactivate marks a fetchable Source active again. Idempotent when already active.
// Manual Sources cannot be reactivated (fetchable Sources only).
func (s *Source) Reactivate() error {
	if s.Type == TypeManual {
		return ErrManualSourceImmutable.New("manual sources cannot be reactivated")
	}
	if s.Active {
		return nil
	}
	s.Active = true
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDetails changes the display name and RSS feed URL of a fetchable Source.
// Manual Sources cannot be edited. Type is never changed.
func (s *Source) UpdateDetails(name, feedURL string) error {
	if s.Type == TypeManual {
		return ErrManualSourceImmutable.New("manual sources cannot be edited")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return apperr.InvalidInput.New("name is required")
	}
	feedURL = strings.TrimSpace(feedURL)
	if s.Type == TypeRSS && feedURL == "" {
		return apperr.InvalidInput.New("rss url is required")
	}
	if s.Type == TypeYouTube && feedURL != "" {
		return apperr.InvalidInput.New("youtube channel cannot be changed")
	}
	s.Name = name
	if s.Type == TypeRSS {
		s.setConfig(map[string]any{"url": feedURL}, feedURL)
	}
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// Rename updates a fetchable Source without revalidating its stored config.
func (s *Source) Rename(name string) error {
	if s.Type == TypeManual {
		return ErrManualSourceImmutable.New("manual sources cannot be edited")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return apperr.InvalidInput.New("name is required")
	}
	s.Name = name
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// ValidateProfileImageChange enforces the Source image invariant.
func (s *Source) ValidateProfileImageChange() error {
	if s.Type == TypeManual {
		return ErrManualSourceImmutable.New("manual sources cannot have profile images")
	}
	return nil
}

func (s *Source) SetProfileImage(id string) error {
	if err := s.ValidateProfileImageChange(); err != nil {
		return err
	}
	s.ProfileImageID = id
	return nil
}
