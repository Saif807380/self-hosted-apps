package model

import (
	"time"

	"github.com/google/uuid"
)

type Book struct {
	ID         uuid.UUID `db:"id"          json:"id"`
	Title      string    `db:"title"       json:"title"`
	Author     string    `db:"author"      json:"author"`
	Rating     *int16    `db:"rating"      json:"rating,omitempty"`
	Review     *string   `db:"review"      json:"review,omitempty"`
	CoverImage *string   `db:"cover_image" json:"coverImage,omitempty"`
	YearsRead  []int16   `db:"-"           json:"yearsRead"`
	Tags       []Tag     `db:"-"           json:"tags"`
	CreatedAt  time.Time `db:"created_at"  json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at"  json:"updatedAt"`
}

type Tag struct {
	ID   uuid.UUID `db:"id"   json:"id"`
	Name string    `db:"name" json:"name"`
}

type Collection struct {
	ID   uuid.UUID `db:"id"   json:"id"`
	Name string    `db:"name" json:"name"`
}

type VideoGame struct {
	ID          uuid.UUID `db:"id"          json:"id"`
	Title       string    `db:"title"       json:"title"`
	Studio      *string   `db:"studio"      json:"studio,omitempty"`
	Rating      *int16    `db:"rating"      json:"rating,omitempty"`
	Review      *string   `db:"review"      json:"review,omitempty"`
	CoverImage  *string   `db:"cover_image" json:"coverImage,omitempty"`
	YearsPlayed []int16   `db:"-"           json:"yearsPlayed"`
	CreatedAt   time.Time `db:"created_at"  json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at"  json:"updatedAt"`
}

type TravelLocation struct {
	ID                 uuid.UUID     `db:"id"                   json:"id"`
	City               string        `db:"city"                 json:"city"`
	Country            string        `db:"country"              json:"country"`
	VisitedFrom        *time.Time    `db:"visited_from"         json:"visitedFrom,omitempty"`
	VisitedTo          *time.Time    `db:"visited_to"           json:"visitedTo,omitempty"`
	PhotoCollectionURL *string       `db:"photo_collection_url" json:"photoCollectionUrl,omitempty"`
	TouristSpots       []TouristSpot `db:"-"                    json:"touristSpots"`
	CreatedAt          time.Time     `db:"created_at"           json:"createdAt"`
	UpdatedAt          time.Time     `db:"updated_at"           json:"updatedAt"`
}

type TouristSpot struct {
	ID          uuid.UUID `db:"id"          json:"id"`
	LocationID  uuid.UUID `db:"location_id" json:"locationId"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
}

type WorkoutType struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	Name      string     `db:"name"       json:"name"`
	SortOrder int16      `db:"sort_order" json:"sortOrder"`
	Exercises []Exercise `db:"-"          json:"exercises"`
	CreatedAt time.Time  `db:"created_at" json:"createdAt"`
}

type Exercise struct {
	ID            uuid.UUID `db:"id"              json:"id"`
	WorkoutTypeID uuid.UUID `db:"workout_type_id" json:"workoutTypeId"`
	Name          string    `db:"name"            json:"name"`
	SortOrder     int16     `db:"sort_order"      json:"sortOrder"`
}

type WorkoutLog struct {
	ID         uuid.UUID `db:"id"          json:"id"`
	ExerciseID uuid.UUID `db:"exercise_id" json:"exerciseId"`
	WeekNumber int16     `db:"week_number" json:"weekNumber"`
	Sets       *int16    `db:"sets"        json:"sets,omitempty"`
	Reps       *string   `db:"reps"        json:"reps,omitempty"`
	WeightKg   *float64  `db:"weight_kg"   json:"weightKg,omitempty"`
	LoggedAt   time.Time `db:"logged_at"   json:"loggedAt"`
}
