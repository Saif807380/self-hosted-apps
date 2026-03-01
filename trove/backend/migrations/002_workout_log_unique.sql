ALTER TABLE workout_logs
  ADD CONSTRAINT workout_logs_exercise_week_unique
  UNIQUE (exercise_id, week_number);
