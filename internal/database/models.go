package database

type EmoteCount struct {
	ID        string `gorm:"primaryKey"`
	GuildID   string `gorm:"primaryKey"`
	ChannelID string `gorm:"primaryKey"`
	Name      string
	Count     int64 `gorm:"default:0"`
	Animated  bool
}

type BackfillCheckpoint struct {
	ChannelID      string `gorm:"primaryKey"`
	GuildID        string `gorm:"primaryKey"`
	LastMessageID  string
	TotalProcessed int64
	TotalEmotes    int64
}
