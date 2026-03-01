package model

import (
	"time"

	"github.com/google/uuid"
)

type Book struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	Rating     *int16    `json:"rating,omitempty"`
	Review     *string   `json:"review,omitempty"`
	CoverImage *string   `json:"coverImage,omitempty"`
	YearsRead  []int16   `json:"yearsRead"`
	Tags       []Tag     `json:"tags"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Tag struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type Collection struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type VideoGame struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Studio      *string   `json:"studio,omitempty"`
	Rating      *int16    `json:"rating,omitempty"`
	Review      *string   `json:"review,omitempty"`
	CoverImage  *string   `json:"coverImage,omitempty"`
	YearsPlayed []int16   `json:"yearsPlayed"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TravelLocation struct {
	ID                 uuid.UUID     `json:"id"`
	City               string        `json:"city"`
	Country            string        `json:"country"`
	VisitedFrom        *time.Time    `json:"visitedFrom,omitempty"`
	VisitedTo          *time.Time    `json:"visitedTo,omitempty"`
	PhotoCollectionURL *string       `json:"photoCollectionUrl,omitempty"`
	TouristSpots       []TouristSpot `json:"touristSpots"`
	CreatedAt          time.Time     `json:"createdAt"`
	UpdatedAt          time.Time     `json:"updatedAt"`
}

type TouristSpot struct {
	ID          uuid.UUID `json:"id"`
	LocationID  uuid.UUID `json:"locationId"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
}

type WorkoutType struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	SortOrder int16      `json:"sortOrder"`
	Exercises []Exercise `json:"exercises"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Exercise struct {
	ID            uuid.UUID `json:"id"`
	WorkoutTypeID uuid.UUID `json:"workoutTypeId"`
	Name          string    `json:"name"`
	SortOrder     int16     `json:"sortOrder"`
}

type WorkoutLog struct {
	ID         uuid.UUID `json:"id"`
	ExerciseID uuid.UUID `json:"exerciseId"`
	WeekNumber int16     `json:"weekNumber"`
	Sets       *int16    `json:"sets,omitempty"`
	Reps       *string   `json:"reps,omitempty"`
	WeightKg   *float64  `json:"weightKg,omitempty"`
	LoggedAt   time.Time `json:"loggedAt"`
}
