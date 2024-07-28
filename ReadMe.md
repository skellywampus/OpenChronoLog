# OpenChronoLog

OpenChronoLog is a compiled Python application that functions as a time clock for teams.

## Features

- Lightweight operation allowing the program to be used on legacy machines.
- User and scan data stored as json files for simple data persistance.
- Visual feedback for successful entries.
- Visual indicator shows the number of users currently clocked in to avoid missed punches.
- Ability to right click the submit button to clock out all users at once.
- Menu options to easily view registered users and clock events.
- Data tables are able to be sorted by each metric.
- Program starts in full screen to allow it to easily run as a kiosk.
- Simple and easy-to-use graphical user interface (GUI).

## Usage

1. **Run the application**
	
2. **Add Users**:
    - Click the button at the bottom titled "Add User"
    - Enter a unique ID for the user.
    - Enter the user's full name.

3. **Manual Entries**:
    - Click the button at the bottom titled "Manual Entry"
	- Enter the user's ID.
	- Enter the time the user clocked in.
	- Enter the time the user clocked out.

4. **Review Data**:
    - Click the button at the bottom titled "Review"
    - "User List" will show all the names and IDs of all users.
	- "Scan Log" will show all clock events. You may select an entry and press the "Delete" button to remove an entry.
	- "Time Data" will allow you to select two dates and will show the amount of time each user has been clocked in during that time window.
	- "Export" will export the Scan Log as a CSV for easy usage in other applications.

5. **Clock In and Out**:
    - Enter your user ID into the main entry box and press "Submit"
	- If you use a card reader or barcode scanner, you will need a third party program that is not provided to input the data automatically.
	- If you use a third party program to handle user ID entries, ensure the IDs are followed by a "Return" event or Enter key press to fully automate the clock process.

6. **Close the Application**:
    - Close the application by pressing the "Close" button in the top right of the window.

## Notes

- Uncompiled python file is included for code auditing and modifications.
- If scan_log.json or user_database.json are not found, the program will automatically create them when any data is entered.
- The scan_log.json and user_database.json files are stored in the same directory as the program's executable and cannot be read by the program if they are moved.
