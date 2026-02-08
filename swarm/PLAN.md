We're making a comprehensive todo-list app called TaskFlow.

The tech stack is NextJS, TypeScript, Tailwind and Convex.dev for the backend.

The features of the platform include:

## Main design decisions

- Make the style look modern and it should have both light and dark mode with a toggle for the user.
- The app should be responsive and work well on mobile devices.
- Use convex-auth for authentication.
- Real-time updates - when tasks are modified, all connected clients should see changes instantly.
- Keyboard shortcuts for power users (e.g., 'n' for new task, 'e' to edit, arrow keys to navigate).

## Dashboard / Home Page

- Overview of all lists and projects
- Quick-add task input at the top
- "Today" view showing tasks due today
- "Upcoming" view showing tasks due in the next 7 days
- "Overdue" view highlighting past-due tasks
- Task completion statistics and streaks

## Lists & Projects

- Users can create multiple lists/projects to organize tasks
- Each list has a name, optional description, and color
- Lists can be reordered via drag-and-drop
- Archive completed lists (soft delete)
- Share lists with other users (view-only or edit permissions)

## Tasks

- Title (required)
- Description (optional, supports markdown)
- Due date and time (optional)
- Priority levels: None, Low, Medium, High, Urgent
- Tags/labels (multiple per task)
- Subtasks/checklist items within a task
- Recurring tasks (daily, weekly, monthly, custom)
- Task attachments (future feature - just add placeholder)
- Assign tasks to other users (for shared lists)

## Task Views

- List view (default)
- Kanban board view (columns by status: To Do, In Progress, Done)
- Calendar view showing tasks by due date
- Filter by: priority, tags, due date, assigned user
- Sort by: due date, priority, created date, alphabetical
- Search across all tasks

## Task Details Page

- Full task details in a slide-out panel or modal
- Activity log showing task history (created, edited, completed, etc.)
- Comments on tasks (for shared lists)
- Time tracking (optional - start/stop timer)

## User Profile & Settings

- Profile page with avatar, name, email
- Notification preferences (email reminders for due tasks)
- Default list setting
- Theme preference (light/dark/system)
- Keyboard shortcut reference

## Sign In/Up

- Use convex-auth to allow sign ins and sign ups
- Support email/password authentication
- Each user has their own workspace of lists and tasks

## Productivity Features

- Daily/weekly task completion stats
- "Focus mode" that shows only the current task
- Pomodoro timer integration (25 min work, 5 min break)
- Daily review prompt showing completed tasks

## Collaboration

- Invite users by email to shared lists
- Real-time presence indicators (who's viewing the list)
- Activity feed for shared lists