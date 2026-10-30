# Unitracker

Unitracker is a student-focused dashboard for tracking subjects, recurring exercise sheets, quizzes, deadlines, and individual exercise scores.

## Current Status

The project is an actively developed prototype. It supports the core workflow for recurring academic work, while some areas may still evolve as the data model and UI are refined.

## What It Does

- Organizes work by subject.
- Creates recurring assignment series for exercise sheets or quizzes.
- Supports compulsory and optional assignments.
- Supports weekly and biweekly recurrence.
- Generates numbered assignment occurrences from:
  - First release date
  - Recurrence interval
  - Repeat-until date
- Calculates release and deadline times for each occurrence.
- Keeps each occurrence as a separate record so previous work and scores remain available.
- Creates separate exercise score rows for every generated occurrence.
- Tracks achieved and possible points for each exercise.
- Stores sheet, solution, and exercise links.
- Filters the dashboard by subject and status.
- Preserves legacy manually created tasks alongside generated assignment occurrences.

For example, a weekly sheet beginning on a Monday and repeating until a selected date becomes a sequence such as:

```text
Exercise Sheet 1
Exercise Sheet 2
Exercise Sheet 3
...
```

Each generated occurrence can contain its own exercise score rows and deadline.

## Architecture

The backend is written in Go and uses Gin for HTTP routing and HTML rendering. HTML templates are embedded into the Go binary.

The main domain layers are:

- `Subject`: a university subject or course.
- `AssignmentSeries`: the recurring definition, including schedule, type, requirement, links, and exercise count.
- `AssignmentOccurrence`: one generated release of a series.
- `Exercise`: an individually scoreable exercise belonging to an occurrence.
- `Task`: the existing flat-task model retained for compatibility with older data.

SQLite is used for persistence through Go's `database/sql` package and the `modernc.org/sqlite` driver. An in-memory store is also available for local tests and development.

## Running With Nix

The repository provides a Nix flake with a development shell.

Run the test suite:

```sh
nix develop --command go test ./...
```

Start the server:

```sh
nix develop --command go run ./cmd/server
```

The server listens on port `8080` by default. The database connection string can be supplied through `CONN_STRING` or the `-conn` flag.

Example:

```sh
CONN_STRING='file:database.db' nix develop --command go run ./cmd/server
```

## Web Interface

Open <http://localhost:8080> after starting the server.

To create recurring work:

1. Create a subject.
2. Select `New assignment series`.
3. Choose the assignment type and requirement.
4. Set the first release date, recurrence interval, and repeat-until date.
5. Set the release and deadline times.
6. Choose how many exercise score rows each release should contain.
7. Add the relevant sheet and solution links.

The dashboard generates the numbered occurrences automatically as time advances. Previous occurrences remain available for historical tracking.

## API and UI Routes

The application exposes JSON-style routes for subjects and tasks, along with UI routes for:

- Creating subjects and tasks.
- Creating recurring assignment series.
- Updating task status.
- Updating task and exercise points.
- Deleting tasks.

The current UI uses server-rendered HTML templates with HTMX for focused updates.

## Testing

Tests cover SQLite task persistence, recurring occurrence generation, exercise creation, exercise score updates, and recurrence date calculations.

Run all tests with:

```sh
nix develop --command go test ./...
```

## Authorship and AI Disclosure

The frontend, including the HTML templates, dashboard layout, styling, and interaction wiring, was generated fully with AI assistance.

The backend was written by a human developer with assistance from AI. The human developer owns the backend design decisions, integration, persistence model, and final code review.
