package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	booksv1 "github.com/saifkazi/trove/backend/gen/trove/v1/books"
	gamesv1 "github.com/saifkazi/trove/backend/gen/trove/v1/games"
	syncv1 "github.com/saifkazi/trove/backend/gen/trove/v1/sync"
	"github.com/saifkazi/trove/backend/gen/trove/v1/sync/syncv1connect"
	travelv1 "github.com/saifkazi/trove/backend/gen/trove/v1/travel"
	workoutsv1 "github.com/saifkazi/trove/backend/gen/trove/v1/workouts"
	trovev1 "github.com/saifkazi/trove/backend/gen/trove/v1"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/service"
)

type syncHandler struct {
	svc *service.SyncService
}

var _ syncv1connect.SyncServiceHandler = (*syncHandler)(nil)

func (h *syncHandler) PushChanges(
	ctx context.Context,
	req *connect.Request[syncv1.PushChangesRequest],
) (*connect.Response[syncv1.PushChangesResponse], error) {
	if req.Msg.Changes == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("changes required"))
	}

	cs, err := changeSetFromProto(req.Msg.Changes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	serverTime, err := h.svc.PushChanges(ctx, cs)
	if err != nil {
		return nil, connectErr(err)
	}

	return connect.NewResponse(&syncv1.PushChangesResponse{
		ServerTime: serverTime.Format(time.RFC3339Nano),
	}), nil
}

func (h *syncHandler) PullChanges(
	ctx context.Context,
	req *connect.Request[syncv1.PullChangesRequest],
) (*connect.Response[syncv1.PullChangesResponse], error) {
	since, err := time.Parse(time.RFC3339Nano, req.Msg.Since)
	if err != nil {
		since, err = time.Parse(time.RFC3339, req.Msg.Since)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid since timestamp: %w", err))
		}
	}

	cs, serverTime, err := h.svc.PullChanges(ctx, since)
	if err != nil {
		return nil, connectErr(err)
	}

	return connect.NewResponse(&syncv1.PullChangesResponse{
		Changes:    changeSetToProto(cs),
		ServerTime: serverTime.Format(time.RFC3339Nano),
	}), nil
}

// --- REST image sync endpoints ---

type imageManifestEntry struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
}

func handleSyncImageUpload(uploadsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "file too large or bad request", http.StatusBadRequest)
			return
		}

		id := r.FormValue("id")
		checksum := r.FormValue("checksum")
		if id == "" || checksum == "" {
			http.Error(w, "id and checksum fields required", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		ext := filepath.Ext(header.Filename)
		if !allowedExtensions[ext] {
			http.Error(w, "unsupported file type", http.StatusBadRequest)
			return
		}

		// Atomic write: write to .tmp, verify checksum, rename
		dest := filepath.Join(uploadsDir, id+ext)
		tmpDest := dest + ".tmp"

		out, err := os.Create(tmpDest)
		if err != nil {
			http.Error(w, "failed to create temp file", http.StatusInternalServerError)
			return
		}

		hasher := sha256.New()
		if _, err := io.Copy(io.MultiWriter(out, hasher), file); err != nil {
			out.Close()
			os.Remove(tmpDest)
			http.Error(w, "failed to write file", http.StatusInternalServerError)
			return
		}
		out.Close()

		computed := hex.EncodeToString(hasher.Sum(nil))
		if computed != checksum {
			os.Remove(tmpDest)
			http.Error(w, fmt.Sprintf("checksum mismatch: expected %s, got %s", checksum, computed), http.StatusBadRequest)
			return
		}

		if err := os.Rename(tmpDest, dest); err != nil {
			os.Remove(tmpDest)
			http.Error(w, "failed to finalize file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": "/uploads/" + id + ext}) //nolint:errcheck
	}
}

func handleSyncImageManifest(uploadsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		entries, err := os.ReadDir(uploadsDir)
		if err != nil {
			http.Error(w, "failed to read uploads", http.StatusInternalServerError)
			return
		}

		manifest := make([]imageManifestEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			ext := filepath.Ext(name)
			if !allowedExtensions[ext] {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			fullPath := filepath.Join(uploadsDir, name)
			checksum, err := fileSHA256(fullPath)
			if err != nil {
				continue
			}

			id := name[:len(name)-len(ext)]
			manifest = append(manifest, imageManifestEntry{
				ID:       id,
				Path:     "/uploads/" + name,
				Checksum: checksum,
				Size:     info.Size(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest) //nolint:errcheck
	}
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// --- Proto <-> Model mapping ---

func changeSetToProto(cs *model.ChangeSet) *syncv1.ChangeSet {
	pb := &syncv1.ChangeSet{}

	for i := range cs.Books {
		pb.Books = append(pb.Books, syncMapBook(&cs.Books[i]))
	}
	for _, t := range cs.Tags {
		pb.Tags = append(pb.Tags, &trovev1.Tag{
			Id: t.ID.String(), Name: t.Name,
			UpdatedAt: t.UpdatedAt.Format(time.RFC3339Nano), Deleted: t.Deleted,
		})
	}
	for _, c := range cs.Collections {
		pb.Collections = append(pb.Collections, &booksv1.Collection{
			Id: c.ID.String(), Name: c.Name,
			UpdatedAt: c.UpdatedAt.Format(time.RFC3339Nano), Deleted: c.Deleted,
		})
	}
	for i := range cs.VideoGames {
		pb.VideoGames = append(pb.VideoGames, syncMapVideoGame(&cs.VideoGames[i]))
	}
	for i := range cs.TravelLocations {
		pb.TravelLocations = append(pb.TravelLocations, syncMapTravelLocation(&cs.TravelLocations[i]))
	}
	for _, s := range cs.TouristSpots {
		pb.TouristSpots = append(pb.TouristSpots, syncMapTouristSpot(&s))
	}
	for _, wt := range cs.WorkoutTypes {
		pb.WorkoutTypes = append(pb.WorkoutTypes, syncMapWorkoutType(&wt))
	}
	for _, e := range cs.Exercises {
		pb.Exercises = append(pb.Exercises, syncMapExercise(&e))
	}
	for _, wl := range cs.WorkoutLogs {
		pb.WorkoutLogs = append(pb.WorkoutLogs, syncMapWorkoutLog(&wl))
	}

	for _, e := range cs.BookYearsRead {
		pb.BookYearsRead = append(pb.BookYearsRead, &syncv1.BookYearRead{BookId: e.BookID.String(), Year: int32(e.Year)})
	}
	for _, e := range cs.BookTags {
		pb.BookTags = append(pb.BookTags, &syncv1.BookTagEntry{BookId: e.BookID.String(), TagId: e.TagID.String()})
	}
	for _, e := range cs.CollectionBooks {
		pb.CollectionBooks = append(pb.CollectionBooks, &syncv1.CollectionBookEntry{CollectionId: e.CollectionID.String(), BookId: e.BookID.String()})
	}
	for _, e := range cs.GameYearsPlayed {
		pb.GameYearsPlayed = append(pb.GameYearsPlayed, &syncv1.GameYearPlayed{GameId: e.GameID.String(), Year: int32(e.Year)})
	}

	return pb
}

func changeSetFromProto(pb *syncv1.ChangeSet) (*model.ChangeSet, error) {
	cs := &model.ChangeSet{}
	var err error

	for _, b := range pb.Books {
		m, err := syncBookFromProto(b)
		if err != nil {
			return nil, fmt.Errorf("book %s: %w", b.Id, err)
		}
		cs.Books = append(cs.Books, *m)
	}
	for _, t := range pb.Tags {
		m, err := syncTagFromProto(t)
		if err != nil {
			return nil, fmt.Errorf("tag %s: %w", t.Id, err)
		}
		cs.Tags = append(cs.Tags, *m)
	}
	for _, c := range pb.Collections {
		m, err := syncCollectionFromProto(c)
		if err != nil {
			return nil, fmt.Errorf("collection %s: %w", c.Id, err)
		}
		cs.Collections = append(cs.Collections, *m)
	}
	for _, g := range pb.VideoGames {
		m, err := syncVideoGameFromProto(g)
		if err != nil {
			return nil, fmt.Errorf("video_game %s: %w", g.Id, err)
		}
		cs.VideoGames = append(cs.VideoGames, *m)
	}
	for _, l := range pb.TravelLocations {
		m, err := syncTravelLocationFromProto(l)
		if err != nil {
			return nil, fmt.Errorf("travel_location %s: %w", l.Id, err)
		}
		cs.TravelLocations = append(cs.TravelLocations, *m)
	}
	for _, s := range pb.TouristSpots {
		m, err := syncTouristSpotFromProto(s)
		if err != nil {
			return nil, fmt.Errorf("tourist_spot %s: %w", s.Id, err)
		}
		cs.TouristSpots = append(cs.TouristSpots, *m)
	}
	for _, wt := range pb.WorkoutTypes {
		m, err := syncWorkoutTypeFromProto(wt)
		if err != nil {
			return nil, fmt.Errorf("workout_type %s: %w", wt.Id, err)
		}
		cs.WorkoutTypes = append(cs.WorkoutTypes, *m)
	}
	for _, e := range pb.Exercises {
		m, err := syncExerciseFromProto(e)
		if err != nil {
			return nil, fmt.Errorf("exercise %s: %w", e.Id, err)
		}
		cs.Exercises = append(cs.Exercises, *m)
	}
	for _, wl := range pb.WorkoutLogs {
		m, err := syncWorkoutLogFromProto(wl)
		if err != nil {
			return nil, fmt.Errorf("workout_log %s: %w", wl.Id, err)
		}
		cs.WorkoutLogs = append(cs.WorkoutLogs, *m)
	}

	// Junction tables
	for _, e := range pb.BookYearsRead {
		id, err := parseUUID(e.BookId)
		if err != nil {
			return nil, err
		}
		cs.BookYearsRead = append(cs.BookYearsRead, model.BookYearRead{BookID: id, Year: int16(e.Year)})
	}
	for _, e := range pb.BookTags {
		bid, err := parseUUID(e.BookId)
		if err != nil {
			return nil, err
		}
		tid, err := parseUUID(e.TagId)
		if err != nil {
			return nil, err
		}
		cs.BookTags = append(cs.BookTags, model.BookTagEntry{BookID: bid, TagID: tid})
	}
	for _, e := range pb.CollectionBooks {
		cid, err := parseUUID(e.CollectionId)
		if err != nil {
			return nil, err
		}
		bid, err := parseUUID(e.BookId)
		if err != nil {
			return nil, err
		}
		cs.CollectionBooks = append(cs.CollectionBooks, model.CollectionBookEntry{CollectionID: cid, BookID: bid})
	}
	for _, e := range pb.GameYearsPlayed {
		id, err := parseUUID(e.GameId)
		if err != nil {
			return nil, err
		}
		cs.GameYearsPlayed = append(cs.GameYearsPlayed, model.GameYearPlayed{GameID: id, Year: int16(e.Year)})
	}

	// Ensure nil slices become empty
	if cs.Books == nil {
		cs.Books = []model.Book{}
	}
	if cs.Tags == nil {
		cs.Tags = []model.Tag{}
	}
	if cs.Collections == nil {
		cs.Collections = []model.Collection{}
	}
	if cs.VideoGames == nil {
		cs.VideoGames = []model.VideoGame{}
	}
	if cs.TravelLocations == nil {
		cs.TravelLocations = []model.TravelLocation{}
	}
	if cs.TouristSpots == nil {
		cs.TouristSpots = []model.TouristSpot{}
	}
	if cs.WorkoutTypes == nil {
		cs.WorkoutTypes = []model.WorkoutType{}
	}
	if cs.Exercises == nil {
		cs.Exercises = []model.Exercise{}
	}
	if cs.WorkoutLogs == nil {
		cs.WorkoutLogs = []model.WorkoutLog{}
	}
	if cs.BookYearsRead == nil {
		cs.BookYearsRead = []model.BookYearRead{}
	}
	if cs.BookTags == nil {
		cs.BookTags = []model.BookTagEntry{}
	}
	if cs.CollectionBooks == nil {
		cs.CollectionBooks = []model.CollectionBookEntry{}
	}
	if cs.GameYearsPlayed == nil {
		cs.GameYearsPlayed = []model.GameYearPlayed{}
	}

	_ = err
	return cs, nil
}

// --- Individual entity mapping: Model → Proto ---

func syncMapBook(b *model.Book) *booksv1.Book {
	pb := &booksv1.Book{
		Id:        b.ID.String(),
		Title:     b.Title,
		Author:    b.Author,
		YearsRead: int32SliceFromInt16(b.YearsRead),
		Tags:      syncMapTags(b.Tags),
		CreatedAt: b.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: b.UpdatedAt.Format(time.RFC3339Nano),
		Deleted:   b.Deleted,
	}
	if b.Rating != nil {
		v := int32(*b.Rating)
		pb.Rating = &v
	}
	if b.Review != nil {
		pb.Review = b.Review
	}
	if b.CoverImage != nil {
		pb.CoverImage = b.CoverImage
	}
	return pb
}

func syncMapTags(tags []model.Tag) []*trovev1.Tag {
	result := make([]*trovev1.Tag, len(tags))
	for i, t := range tags {
		result[i] = &trovev1.Tag{
			Id: t.ID.String(), Name: t.Name,
			UpdatedAt: t.UpdatedAt.Format(time.RFC3339Nano), Deleted: t.Deleted,
		}
	}
	return result
}

func syncMapVideoGame(g *model.VideoGame) *gamesv1.VideoGame {
	pb := &gamesv1.VideoGame{
		Id:          g.ID.String(),
		Title:       g.Title,
		YearsPlayed: int32SliceFromInt16(g.YearsPlayed),
		CreatedAt:   g.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   g.UpdatedAt.Format(time.RFC3339Nano),
		Deleted:     g.Deleted,
	}
	if g.Studio != nil {
		pb.Studio = g.Studio
	}
	if g.Rating != nil {
		v := int32(*g.Rating)
		pb.Rating = &v
	}
	if g.Review != nil {
		pb.Review = g.Review
	}
	if g.CoverImage != nil {
		pb.CoverImage = g.CoverImage
	}
	return pb
}

func syncMapTravelLocation(l *model.TravelLocation) *travelv1.TravelLocation {
	pb := &travelv1.TravelLocation{
		Id:      l.ID.String(),
		City:    l.City,
		Country: l.Country,
		CreatedAt: l.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: l.UpdatedAt.Format(time.RFC3339Nano),
		Deleted:   l.Deleted,
	}
	if l.VisitedFrom != nil {
		v := l.VisitedFrom.Format("2006-01-02")
		pb.VisitedFrom = &v
	}
	if l.VisitedTo != nil {
		v := l.VisitedTo.Format("2006-01-02")
		pb.VisitedTo = &v
	}
	if l.PhotoCollectionURL != nil {
		pb.PhotoCollectionUrl = l.PhotoCollectionURL
	}
	return pb
}

func syncMapTouristSpot(s *model.TouristSpot) *travelv1.TouristSpot {
	pb := &travelv1.TouristSpot{
		Id:         s.ID.String(),
		LocationId: s.LocationID.String(),
		Name:       s.Name,
		UpdatedAt:  s.UpdatedAt.Format(time.RFC3339Nano),
		Deleted:    s.Deleted,
	}
	if s.Description != nil {
		pb.Description = s.Description
	}
	return pb
}

func syncMapWorkoutType(wt *model.WorkoutType) *workoutsv1.WorkoutType {
	return &workoutsv1.WorkoutType{
		Id:        wt.ID.String(),
		Name:      wt.Name,
		SortOrder: int32(wt.SortOrder),
		CreatedAt: wt.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: wt.UpdatedAt.Format(time.RFC3339Nano),
		Deleted:   wt.Deleted,
	}
}

func syncMapExercise(e *model.Exercise) *workoutsv1.Exercise {
	return &workoutsv1.Exercise{
		Id:            e.ID.String(),
		WorkoutTypeId: e.WorkoutTypeID.String(),
		Name:          e.Name,
		SortOrder:     int32(e.SortOrder),
		UpdatedAt:     e.UpdatedAt.Format(time.RFC3339Nano),
		Deleted:       e.Deleted,
	}
}

func syncMapWorkoutLog(wl *model.WorkoutLog) *workoutsv1.WorkoutLog {
	pb := &workoutsv1.WorkoutLog{
		Id:         wl.ID.String(),
		ExerciseId: wl.ExerciseID.String(),
		WeekNumber: int32(wl.WeekNumber),
		LoggedAt:   wl.LoggedAt.Format(time.RFC3339Nano),
		UpdatedAt:  wl.UpdatedAt.Format(time.RFC3339Nano),
		Deleted:    wl.Deleted,
	}
	if wl.Sets != nil {
		v := int32(*wl.Sets)
		pb.Sets = &v
	}
	if wl.Reps != nil {
		pb.Reps = wl.Reps
	}
	if wl.WeightKg != nil {
		pb.WeightKg = wl.WeightKg
	}
	return pb
}

// --- Individual entity mapping: Proto → Model ---

func syncBookFromProto(pb *booksv1.Book) (*model.Book, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, pb.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)

	b := &model.Book{
		ID:        id,
		Title:     pb.Title,
		Author:    pb.Author,
		Rating:    int16PtrFromInt32(pb.Rating),
		Review:    pb.Review,
		CoverImage: pb.CoverImage,
		YearsRead: int16SliceFromInt32(pb.YearsRead),
		Deleted:   pb.Deleted,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	for _, t := range pb.Tags {
		tid, err := parseUUID(t.Id)
		if err != nil {
			return nil, err
		}
		tagUpdated, _ := time.Parse(time.RFC3339Nano, t.UpdatedAt)
		b.Tags = append(b.Tags, model.Tag{ID: tid, Name: t.Name, Deleted: t.Deleted, UpdatedAt: tagUpdated})
	}
	if b.Tags == nil {
		b.Tags = []model.Tag{}
	}
	if b.YearsRead == nil {
		b.YearsRead = []int16{}
	}
	return b, nil
}

func syncTagFromProto(pb *trovev1.Tag) (*model.Tag, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)
	return &model.Tag{ID: id, Name: pb.Name, Deleted: pb.Deleted, UpdatedAt: updatedAt}, nil
}

func syncCollectionFromProto(pb *booksv1.Collection) (*model.Collection, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)
	return &model.Collection{ID: id, Name: pb.Name, Deleted: pb.Deleted, UpdatedAt: updatedAt}, nil
}

func syncVideoGameFromProto(pb *gamesv1.VideoGame) (*model.VideoGame, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, pb.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)

	g := &model.VideoGame{
		ID:          id,
		Title:       pb.Title,
		Studio:      pb.Studio,
		Rating:      int16PtrFromInt32(pb.Rating),
		Review:      pb.Review,
		CoverImage:  pb.CoverImage,
		YearsPlayed: int16SliceFromInt32(pb.YearsPlayed),
		Deleted:     pb.Deleted,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if g.YearsPlayed == nil {
		g.YearsPlayed = []int16{}
	}
	return g, nil
}

func syncTravelLocationFromProto(pb *travelv1.TravelLocation) (*model.TravelLocation, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, pb.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)

	l := &model.TravelLocation{
		ID:                 id,
		City:               pb.City,
		Country:            pb.Country,
		PhotoCollectionURL: pb.PhotoCollectionUrl,
		TouristSpots:       []model.TouristSpot{},
		Deleted:            pb.Deleted,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}
	if pb.VisitedFrom != nil {
		t, _ := time.Parse("2006-01-02", *pb.VisitedFrom)
		l.VisitedFrom = &t
	}
	if pb.VisitedTo != nil {
		t, _ := time.Parse("2006-01-02", *pb.VisitedTo)
		l.VisitedTo = &t
	}
	return l, nil
}

func syncTouristSpotFromProto(pb *travelv1.TouristSpot) (*model.TouristSpot, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	locID, err := parseUUID(pb.LocationId)
	if err != nil {
		return nil, err
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)
	return &model.TouristSpot{
		ID: id, LocationID: locID, Name: pb.Name, Description: pb.Description,
		Deleted: pb.Deleted, UpdatedAt: updatedAt,
	}, nil
}

func syncWorkoutTypeFromProto(pb *workoutsv1.WorkoutType) (*model.WorkoutType, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, pb.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)
	return &model.WorkoutType{
		ID: id, Name: pb.Name, SortOrder: int16(pb.SortOrder),
		Exercises: []model.Exercise{}, Deleted: pb.Deleted,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func syncExerciseFromProto(pb *workoutsv1.Exercise) (*model.Exercise, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	wtID, err := parseUUID(pb.WorkoutTypeId)
	if err != nil {
		return nil, err
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)
	return &model.Exercise{
		ID: id, WorkoutTypeID: wtID, Name: pb.Name, SortOrder: int16(pb.SortOrder),
		Deleted: pb.Deleted, UpdatedAt: updatedAt,
	}, nil
}

func syncWorkoutLogFromProto(pb *workoutsv1.WorkoutLog) (*model.WorkoutLog, error) {
	id, err := parseUUID(pb.Id)
	if err != nil {
		return nil, err
	}
	exID, err := parseUUID(pb.ExerciseId)
	if err != nil {
		return nil, err
	}
	loggedAt, _ := time.Parse(time.RFC3339Nano, pb.LoggedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, pb.UpdatedAt)

	wl := &model.WorkoutLog{
		ID: id, ExerciseID: exID, WeekNumber: int16(pb.WeekNumber),
		Deleted: pb.Deleted, LoggedAt: loggedAt, UpdatedAt: updatedAt,
	}
	if pb.Sets != nil {
		v := int16(*pb.Sets)
		wl.Sets = &v
	}
	if pb.Reps != nil {
		wl.Reps = pb.Reps
	}
	if pb.WeightKg != nil {
		wl.WeightKg = pb.WeightKg
	}
	return wl, nil
}
