package database

type EmoteCount struct {
	GuildID   string `gorm:"primaryKey"`
	EmoteID   string `gorm:"primaryKey"`
	EmoteName string
	Count     int64 `gorm:"default:0"`
}
