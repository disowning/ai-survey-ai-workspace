package models

import "time"

type Profile struct {
	ID          int64     `json:"id"`
	ProfileKey  string    `json:"profile_key"`
	ProfileName *string   `json:"profile_name,omitempty"`
	Remark      *string   `json:"remark,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SurveySite struct {
	ID        int64     `json:"id"`
	SiteKey   string    `json:"site_key"`
	SiteName  *string   `json:"site_name,omitempty"`
	Domain    *string   `json:"domain,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProfileSite struct {
	ID               int64     `json:"id"`
	ProfileID        int64     `json:"profile_id"`
	SiteKey          string    `json:"site_key"`
	SiteAccountEmail *string   `json:"site_account_email,omitempty"`
	SiteRemark       *string   `json:"site_remark,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Survey struct {
	ID          int64     `json:"id"`
	ProfileID   int64     `json:"profile_id"`
	SiteKey     string    `json:"site_key"`
	SurveyTitle *string   `json:"survey_title,omitempty"`
	SurveyURL   *string   `json:"survey_url,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PageSnapshot struct {
	ID           int64     `json:"id"`
	ProfileID    int64     `json:"profile_id"`
	SiteKey      string    `json:"site_key"`
	SurveyID     *int64    `json:"survey_id,omitempty"`
	URL          *string   `json:"url,omitempty"`
	PageTitle    *string   `json:"page_title,omitempty"`
	QuestionText *string   `json:"question_text,omitempty"`
	OptionsText  *string   `json:"options_text,omitempty"`
	PageText     *string   `json:"page_text,omitempty"`
	Language     *string   `json:"language,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Note struct {
	ID             int64     `json:"id"`
	ProfileID      int64     `json:"profile_id"`
	SiteKey        string    `json:"site_key"`
	SurveyID       *int64    `json:"survey_id,omitempty"`
	PageSnapshotID *int64    `json:"page_snapshot_id,omitempty"`
	NoteText       string    `json:"note_text"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Translation struct {
	ID             int64     `json:"id"`
	ProfileID      int64     `json:"profile_id"`
	SiteKey        string    `json:"site_key"`
	SourceText     string    `json:"source_text"`
	TranslatedText string    `json:"translated_text"`
	SourceLang     *string   `json:"source_lang,omitempty"`
	TargetLang     string    `json:"target_lang"`
	CreatedAt      time.Time `json:"created_at"`
}

type AIConversation struct {
	ID          int64     `json:"id"`
	ProfileID   int64     `json:"profile_id"`
	SiteKey     string    `json:"site_key"`
	SurveyID    *int64    `json:"survey_id,omitempty"`
	UserMessage string    `json:"user_message"`
	AIMessage   string    `json:"ai_message"`
	CreatedAt   time.Time `json:"created_at"`
}

type KnowledgeChunk struct {
	ID         int64     `json:"id"`
	ProfileID  int64     `json:"profile_id"`
	SiteKey    string    `json:"site_key"`
	SurveyID   *int64    `json:"survey_id,omitempty"`
	Scope      string    `json:"scope"`
	SourceType string    `json:"source_type"`
	SourceID   *int64    `json:"source_id,omitempty"`
	ChunkText  string    `json:"chunk_text"`
	CreatedAt  time.Time `json:"created_at"`
}

type SitePersona struct {
	ID           int64     `json:"id"`
	ProfileID    int64     `json:"profile_id"`
	SiteKey      string    `json:"site_key"`
	Category     string    `json:"category"`
	PersonaKey   string    `json:"persona_key"`
	PersonaValue string    `json:"persona_value"`
	Confidence   float64   `json:"confidence"`
	SourceType   string    `json:"source_type"`
	SourceID     *int64    `json:"source_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AnswerRecord struct {
	ID              int64     `json:"id"`
	ProfileID       int64     `json:"profile_id"`
	SiteKey         string    `json:"site_key"`
	SurveyID        *int64    `json:"survey_id,omitempty"`
	PageSnapshotID  *int64    `json:"page_snapshot_id,omitempty"`
	QuestionText    string    `json:"question_text"`
	OptionsText     *string   `json:"options_text,omitempty"`
	SuggestedAnswer *string   `json:"suggested_answer,omitempty"`
	FinalAnswer     string    `json:"final_answer"`
	Reason          *string   `json:"reason,omitempty"`
	PersonaMatched  *string   `json:"persona_matched,omitempty"`
	Confidence      float64   `json:"confidence"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
