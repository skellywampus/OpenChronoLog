![Main Menu Screenshot](https://github.com/skellywampus/OpenChronoLog/blob/main/openChronoLogScreenshot.png "Main Menu")

# OpenChronoLog

OpenChronoLog is a lightweight, user-friendly time-tracking application designed to help users efficiently clock in and out. It supports adding new users, reviewing data, making manual entries, and more. This application is perfect for small businesses, teams, or individuals looking to manage their work hours and user data.

## Features

- **Clock In/Out:** Users can clock in and out with their unique User ID.
- **User Management:** Easily add new users or review a list of registered users.
- **Manual Entries:** Make manual time entries for missed clock-ins or clock-outs.
- **Data Review:** View clocked-in users, scan logs, and total time worked.
- **Export Options:** Export time data in CSV format for easy review and reporting.
- **Clock Out All Users:** Conveniently clock out all users at the end of the day.

## System Requirements

- **Operating System:** Windows 10 or Windows 11
- **Disk Space:** Minimal; the application is lightweight
- **Permissions:** Ability to run executable files

## Installation

1. **Download the Application:**
   - Obtain the `.exe` file from the releases page.
   
2. **Save the Application:**
   - Save the `.exe` file in a folder where you have read/write permissions (e.g., Desktop or Documents folder).

3. **First Launch:**
   - Upon the first launch, the application will create two data files in the same directory:
     - `user_database.json`
     - `scan_log.json`
   - These files store user and time data, so do not move or delete them.

## How to Use

### Clocking In/Out
1. Enter your User ID in the text box.
2. Press `Enter` or click the submit button.
3. The system will automatically update your status (clocked in or out).

### Adding a New User
1. Click on the **Add User** button.
2. Enter the User ID and full name.
3. Confirm the addition.

### Manual Time Entries
1. Use the **Manual Entry** option.
2. Enter the User ID, clock-in, and clock-out times.
3. Confirm the entry.

### Reviewing Data
1. Click the **Review** button.
2. View the User List, Scan Log, or calculate total hours worked.
3. Export data as needed.

### Clock Out All Users
1. Right-click on the Submit button to initiate the clock-out-all function.
2. Confirm the action.

## Exporting Data
Export all your time and user data to a CSV file by clicking the **Export Data** option in the Review section. You can then open the file with any spreadsheet software for further analysis.

## Keyboard Shortcuts

- **Fullscreen Mode:** Press `F11` to toggle fullscreen.
- **Submit User ID:** Press `Enter` after typing your User ID.
- **Exit Application:** Press `Escape` to exit fullscreen or click the X to close the window.

## Troubleshooting

- **User Not Found:** Verify the User ID and check if the user is registered.
- **Invalid Time Format:** Ensure time entries follow the `HH:MM:SS` format.
- **Application Won’t Launch:** Make sure you have the necessary permissions or try running the application as an administrator.

## License

This project is licensed under the GNU GPL License. See the LICENSE file for more information.
