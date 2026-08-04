package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"liike_app/internal/domain"
)

// --- User ---

func TestUser_JSONMarshal_OmitsPasswordHash(t *testing.T) {
	u := domain.User{
		ID:           "user-1",
		Email:        "test@example.com",
		PasswordHash: "secret-hash",
		Name:         "Testi Käyttäjä",
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal(User) epäonnistui: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal epäonnistui: %v", err)
	}

	if _, exists := result["password_hash"]; exists {
		t.Error("password_hash ei saisi näkyä JSON-tulosteessa (tag: json:\"-\")")
	}
	if result["email"] != "test@example.com" {
		t.Errorf("odotettu email 'test@example.com', saatiin '%v'", result["email"])
	}
	if result["name"] != "Testi Käyttäjä" {
		t.Errorf("odotettu name 'Testi Käyttäjä', saatiin '%v'", result["name"])
	}
}

func TestUser_ZeroValue(t *testing.T) {
	var u domain.User
	if u.ID != "" || u.Email != "" || u.Name != "" {
		t.Error("User-nollaarvon kentät eivät ole tyhjiä merkkijonoja")
	}
}

// --- ExerciseCategory ---

func TestExerciseCategory_Constants(t *testing.T) {
	cases := []struct {
		got  domain.ExerciseCategory
		want string
	}{
		{domain.CategoryCardio, "cardio"},
		{domain.CategoryStrength, "strength"},
		{domain.CategoryArchery, "archery"},
		{domain.CategoryFlexibility, "flexibility"},
	}

	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("ExerciseCategory-vakio: odotettu '%s', saatiin '%s'", tc.want, tc.got)
		}
	}
}

func TestExerciseType_JSONMarshal(t *testing.T) {
	et := domain.ExerciseType{
		ID:          "et-1",
		Name:        "Juoksu",
		Category:    domain.CategoryCardio,
		Description: "Juoksulenkki",
		CreatedAt:   time.Now(),
	}

	data, err := json.Marshal(et)
	if err != nil {
		t.Fatalf("json.Marshal(ExerciseType) epäonnistui: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal epäonnistui: %v", err)
	}

	if result["name"] != "Juoksu" {
		t.Errorf("odotettu name 'Juoksu', saatiin '%v'", result["name"])
	}
	if result["category"] != "cardio" {
		t.Errorf("odotettu category 'cardio', saatiin '%v'", result["category"])
	}
}

// --- Workout ---

func TestWorkout_OptionalFieldsNil(t *testing.T) {
	w := domain.Workout{
		ID:             "wk-1",
		UserID:         "user-1",
		ExerciseTypeID: "et-1",
		StartTime:      time.Now(),
	}

	if w.EndTime != nil {
		t.Error("EndTime pitäisi olla nil oletuksena")
	}
	if w.DurationSeconds != nil {
		t.Error("DurationSeconds pitäisi olla nil oletuksena")
	}
	if w.DistanceKm != nil {
		t.Error("DistanceKm pitäisi olla nil oletuksena")
	}
	if w.AvgHeartRate != nil {
		t.Error("AvgHeartRate pitäisi olla nil oletuksena")
	}
	if w.CaloriesBurned != nil {
		t.Error("CaloriesBurned pitäisi olla nil oletuksena")
	}
	if w.Notes != nil {
		t.Error("Notes pitäisi olla nil oletuksena")
	}
}

func TestWorkout_JSONMarshal_OmitsEmptyRelations(t *testing.T) {
	w := domain.Workout{
		ID:             "wk-1",
		UserID:         "user-1",
		ExerciseTypeID: "et-1",
		StartTime:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("json.Marshal(Workout) epäonnistui: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal epäonnistui: %v", err)
	}

	// exercise_type on omitempty, ei pitäisi näkyä kun nil
	if _, exists := result["exercise_type"]; exists {
		t.Error("exercise_type ei saisi näkyä JSON:ssa kun se on nil (omitempty)")
	}
	if result["user_id"] != "user-1" {
		t.Errorf("odotettu user_id 'user-1', saatiin '%v'", result["user_id"])
	}
}

func TestWorkoutSet_Fields(t *testing.T) {
	reps := 10
	weight := 80.5

	ws := domain.WorkoutSet{
		ID:        "ws-1",
		WorkoutID: "wk-1",
		SetNumber: 1,
		Reps:      &reps,
		WeightKg:  &weight,
		CreatedAt: time.Now(),
	}

	if *ws.Reps != 10 {
		t.Errorf("odotettu Reps 10, saatiin %d", *ws.Reps)
	}
	if *ws.WeightKg != 80.5 {
		t.Errorf("odotettu WeightKg 80.5, saatiin %f", *ws.WeightKg)
	}
	if ws.SetNumber != 1 {
		t.Errorf("odotettu SetNumber 1, saatiin %d", ws.SetNumber)
	}
}

// --- ArcheryScore ---

func TestArcheryScore_IsXDefault(t *testing.T) {
	score := domain.ArcheryScore{
		ID:          "as-1",
		WorkoutID:   "wk-1",
		EndNumber:   1,
		ArrowNumber: 1,
		ScoreValue:  10,
		IsX:         false,
		CreatedAt:   time.Now(),
	}

	if score.IsX {
		t.Error("IsX pitäisi olla false oletuksena")
	}
	if score.ScoreValue != 10 {
		t.Errorf("odotettu ScoreValue 10, saatiin %d", score.ScoreValue)
	}
}

func TestArcheryScore_JSONMarshal(t *testing.T) {
	score := domain.ArcheryScore{
		ID:          "as-1",
		WorkoutID:   "wk-1",
		EndNumber:   2,
		ArrowNumber: 3,
		ScoreValue:  9,
		IsX:         true,
		CreatedAt:   time.Now(),
	}

	data, err := json.Marshal(score)
	if err != nil {
		t.Fatalf("json.Marshal(ArcheryScore) epäonnistui: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal epäonnistui: %v", err)
	}

	if result["end_number"] != float64(2) {
		t.Errorf("odotettu end_number 2, saatiin %v", result["end_number"])
	}
	if result["is_x"] != true {
		t.Errorf("odotettu is_x true, saatiin %v", result["is_x"])
	}
	if result["score_value"] != float64(9) {
		t.Errorf("odotettu score_value 9, saatiin %v", result["score_value"])
	}
}
