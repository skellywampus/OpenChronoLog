import tkinter as tk
from tkinter import ttk, simpledialog, messagebox, filedialog
from datetime import datetime, timedelta
import json
import os
import csv

USER_DB_FILE = "user_database.json"
SCAN_LOG_FILE = "scan_log.json"

def load_data():
    global user_database, scan_log
    if os.path.exists(USER_DB_FILE):
        with open(USER_DB_FILE, "r") as f:
            user_database = json.load(f)
    else:
        user_database = []

    if os.path.exists(SCAN_LOG_FILE):
        with open(SCAN_LOG_FILE, "r") as f:
            scan_log = json.load(f)
    else:
        scan_log = []

def save_data():
    with open(USER_DB_FILE, "w") as f:
        json.dump(user_database, f)
    with open(SCAN_LOG_FILE, "w") as f:
        json.dump(scan_log, f)

user_database = []
scan_log = []

def find_user_by_id(user_id):
    for user in user_database:
        if user["id"] == user_id:
            return user
    return None

def update_clocked_in_count():
    count = sum(1 for user in user_database if user["status"] == "clocked_in")
    clocked_in_label.config(text=f"{count} users currently clocked in")

def submit_action():
    try:
        user_id = entry.get().strip()
        if not user_id:
            raise ValueError("User ID cannot be empty")

        user = find_user_by_id(user_id)
        if user:
            handle_scan(user)
        else:
            raise ValueError(f"User with ID {user_id} not found")
        
        save_data()
        update_clocked_in_count()
        success_label.place(relx=0.5, rely=0.65, anchor="center")
        window.after(2000, lambda: success_label.place_forget())  # Hide after 2 seconds

    except ValueError as e:
        messagebox.showerror("Error", str(e))

    entry.delete(0, tk.END)
    entry.focus_set()

def handle_scan(user):
    current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    if user["status"] == "clocked_in":
        user["status"] = "clocked_out"
        calculate_duration(user["id"], current_time)
    else:
        user["status"] = "clocked_in"

    scan_log.append({"user_id": user["id"], "name": user["name"], "time": current_time, "status": user["status"], "entry_type": "Scan"})

def calculate_duration(user_id, clock_out_time):
    clock_in_time = next((entry["time"] for entry in scan_log[::-1] if entry["user_id"] == user_id and entry["status"] == "clocked_in"), None)

    if clock_in_time:
        clock_in_time = datetime.strptime(clock_in_time, "%Y-%m-%d %H:%M:%S")
        clock_out_time = datetime.strptime(clock_out_time, "%Y-%m-%d %H:%M:%S")

        duration = clock_out_time - clock_in_time
        duration_str = str(duration).split(".")[0]
        print(f"{user_id} was clocked in for {duration_str}")

def toggle_fullscreen(event=None):
    fullscreen = not window.attributes("-fullscreen")
    window.attributes("-fullscreen", fullscreen)
    if not fullscreen:
        window.geometry(f"{screen_width}x{screen_height}+0+0")  # Restore window size and position

def exit_fullscreen(event=None):
    window.attributes("-fullscreen", False)

def manual_entry_action():
    try:
        user_id = simpledialog.askstring("Manual Entry", "Enter User ID:", parent=window)
        if user_id is None:
            return
        time_in = simpledialog.askstring("Manual Entry", "Enter Time In (HH:MM:SS):", parent=window)
        if time_in is None:
            return
        time_out = simpledialog.askstring("Manual Entry", "Enter Time Out (HH:MM:SS):", parent=window)
        if time_out is None:
            return

        if not user_id or not time_in or not time_out:
            raise ValueError("User ID, Time In, and Time Out cannot be empty")

        current_date = datetime.now().strftime('%Y-%m-%d')
        time_in_full = f"{current_date} {time_in}"
        time_out_full = f"{current_date} {time_out}"

        user = find_user_by_id(user_id)
        if user:
            user["status"] = "clocked_out"
            calculate_duration(user["id"], time_out_full)
            print(f"{user['name']} manually clocked in at {time_in_full} and clocked out at {time_out_full}")
            scan_log.append({"user_id": user["id"], "name": user["name"], "time": time_in_full, "status": "clocked_in", "entry_type": "Manual"})
            scan_log.append({"user_id": user["id"], "name": user["name"], "time": time_out_full, "status": "clocked_out", "entry_type": "Manual"})
            save_data()
        else:
            raise ValueError(f"User with ID {user_id} not found")

    except ValueError as e:
        messagebox.showerror("Error", str(e), parent=window)

def add_user_action():
    try:
        user_id = simpledialog.askstring("Add User", "Enter User ID:", parent=window)
        if user_id is None:
            return
        user_name = simpledialog.askstring("Add User", "Enter User Name:", parent=window)
        if user_name is None:
            return
        if not user_id or not user_name:
            raise ValueError("User ID and User Name cannot be empty")

        if find_user_by_id(user_id):
            raise ValueError("User with this ID already exists")

        user_database.append({"id": user_id, "name": user_name, "status": "clocked_out"})
        save_data()
        print(f"Added user: {user_name} with ID: {user_id}")

    except ValueError as e:
        messagebox.showerror("Error", str(e), parent=window)

def review_action():
    review_window = tk.Toplevel(window)
    review_window.title("Review Data")

    def show_user_list():
        display_data("User List", user_database, ["id", "name"])

    def show_scan_log():
        log_window = display_data("Scan Log", scan_log, ["user_id", "name", "time", "status", "entry_type"])
        
        def delete_entry():
            selected_item = tree.selection()
            if not selected_item:
                messagebox.showerror("Error", "No item selected")
                return

            confirmation = messagebox.askyesno("Confirm Delete", "Are you sure you want to delete the selected entry?")
            if confirmation:
                item = tree.item(selected_item)
                entry_values = item["values"]
                scan_log[:] = [entry for entry in scan_log if not (
                    entry["user_id"] == entry_values[0] and
                    entry["name"] == entry_values[1] and
                    entry["time"] == entry_values[2] and
                    entry["status"] == entry_values[3] and
                    entry["entry_type"] == entry_values[4]
                )]
                tree.delete(selected_item)
                save_data()
                messagebox.showinfo("Success", "Entry deleted successfully")

        delete_button = ttk.Button(log_window, text="Delete Entry", command=delete_entry)
        delete_button.pack(pady=10)

    def show_time_data():
        time_data_window = tk.Toplevel(review_window)
        time_data_window.title("Time Data")

        tk.Label(time_data_window, text="Select Start Date:").grid(row=0, column=0)
        tk.Label(time_data_window, text="Select End Date:").grid(row=1, column=0)

        start_month = ttk.Combobox(time_data_window, values=[f"{i:02d}" for i in range(1, 13)], width=5)
        start_month.grid(row=0, column=1)
        start_month.set("01")

        start_year = ttk.Combobox(time_data_window, values=[str(i) for i in range(2000, 2051)], width=5)
        start_year.grid(row=0, column=2)
        start_year.set(str(datetime.now().year))

        end_month = ttk.Combobox(time_data_window, values=[f"{i:02d}" for i in range(1, 13)], width=5)
        end_month.grid(row=1, column=1)
        end_month.set("01")

        end_year = ttk.Combobox(time_data_window, values=[str(i) for i in range(2000, 2051)], width=5)
        end_year.grid(row=1, column=2)
        end_year.set(str(datetime.now().year))

        def show_today():
            end_month.set(datetime.now().strftime('%m'))
            end_year.set(datetime.now().strftime('%Y'))

        ttk.Button(time_data_window, text="Today", command=show_today).grid(row=1, column=3)

        def calculate_time_data():
            start_date = f"{start_year.get()}-{start_month.get()}-01"
            end_date = f"{end_year.get()}-{end_month.get()}-{(datetime(int(end_year.get()), int(end_month.get()) + 1, 1) - timedelta(days=1)).day}"

            total_time = {}
            for entry in scan_log:
                if start_date <= entry["time"].split(" ")[0] <= end_date:
                    user_id = entry["user_id"]
                    if user_id not in total_time:
                        total_time[user_id] = timedelta()
                    if entry["status"] == "clocked_in":
                        clock_in_time = datetime.strptime(entry["time"], "%Y-%m-%d %H:%M:%S")
                    elif entry["status"] == "clocked_out":
                        clock_out_time = datetime.strptime(entry["time"], "%Y-%m-%d %H:%M:%S")
                        total_time[user_id] += clock_out_time - clock_in_time

            time_data_list = [{"user_id": user_id, "name": find_user_by_id(user_id)["name"], "time_worked": str(total_time[user_id])} for user_id in total_time]
            display_data("Time Data", time_data_list, ["user_id", "name", "time_worked"])

        ttk.Button(time_data_window, text="Calculate", command=calculate_time_data).grid(row=2, column=0, columnspan=4, pady=10)

    def export_data():
        file_path = filedialog.asksaveasfilename(defaultextension=".csv", filetypes=[("CSV files", "*.csv"), ("All files", "*.*")])
        if file_path:
            try:
                with open(file_path, mode='w', newline='') as file:
                    writer = csv.writer(file)
                    writer.writerow(["User ID", "Name", "Time", "Status", "Entry Type"])
                    for entry in scan_log:
                        writer.writerow([entry["user_id"], entry["name"], entry["time"], entry["status"], entry["entry_type"]])
                messagebox.showinfo("Export Successful", f"Data successfully exported to {file_path}")
            except Exception as e:
                messagebox.showerror("Export Failed", f"An error occurred while exporting data: {str(e)}")

    ttk.Button(review_window, text="User List", command=show_user_list).pack(pady=10)
    ttk.Button(review_window, text="Scan Log", command=show_scan_log).pack(pady=10)
    ttk.Button(review_window, text="Time Data", command=show_time_data).pack(pady=10)
    ttk.Button(review_window, text="Export Data", command=export_data).pack(pady=10)

def clock_out_all_users(event=None):
    confirm = messagebox.askyesno("Confirm Clock Out", "Are you sure you want to clock out all users?")
    if confirm:
        current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        for user in user_database:
            if user["status"] == "clocked_in":
                user["status"] = "clocked_out"
                scan_log.append({
                    "user_id": user["id"],
                    "name": user["name"],
                    "time": current_time,
                    "status": "clocked_out",
                    "entry_type": "Manual"
                })
        save_data()
        update_clocked_in_count()
        messagebox.showinfo("All Users Clocked Out", "All users have been clocked out successfully.")

def display_data(title, data, columns):
    data_window = tk.Toplevel(window)
    data_window.title(title)

    global tree
    tree = ttk.Treeview(data_window, columns=columns, show='headings')
    for col in columns:
        tree.heading(col, text=col)
        tree.column(col, minwidth=0, width=100, stretch=tk.NO)

    for item in data:
        values = [item[col] for col in columns]
        tree.insert("", tk.END, values=values)

    tree.pack(expand=True, fill=tk.BOTH)
    treeview_sortable(tree, columns)

    return data_window

def treeview_sortable(tree, columns):
    for col in columns:
        tree.heading(col, text=col, command=lambda _col=col: sort_treeview_column(tree, _col, False))

def sort_treeview_column(tree, col, reverse):
    l = [(tree.set(k, col), k) for k in tree.get_children('')]
    l.sort(reverse=reverse)

    for index, (val, k) in enumerate(l):
        tree.move(k, '', index)

    tree.heading(col, command=lambda: sort_treeview_column(tree, col, not reverse))

def close_application():
    if messagebox.askokcancel("Quit", "Do you want to quit?"):
        window.destroy()

def confirm_quit():
    if messagebox.askokcancel("Quit", "Do you want to quit?"):
        window.destroy()

dark_background = "#2E2E2E"
light_text = "#FFFFFF"
button_text_color = "#000000"
button_color = "#404040"
button_hover = "#505050"
text_box_color = "#808080"

load_data()

window = tk.Tk()
window.title("OpenChronoLog")
window.configure(bg=dark_background)

screen_width = window.winfo_screenwidth()
screen_height = window.winfo_screenheight()
window.geometry(f"{screen_width}x{screen_height}+0+0")

window.attributes("-fullscreen", True)

# Set up key bindings for fullscreen toggle
window.bind("<F11>", toggle_fullscreen)
window.bind("<Escape>", exit_fullscreen)

style = ttk.Style()
style.configure("TButton", background=button_color, foreground=button_text_color, font=("Arial", "12"))
style.map("TButton", background=[("active", button_hover)])

appName = tk.Label(window, text="OpenChronoLog", font=("Arial", 55), bg=dark_background, fg=light_text)
label = tk.Label(window, text="Enter user ID", font=("Arial", 28), bg=dark_background, fg=light_text)
entry = tk.Entry(window, width=30, bg=text_box_color, fg=light_text, insertbackground=light_text, relief=tk.FLAT)

submit_button = ttk.Button(window, text="Submit", command=submit_action)
entry.bind("<Return>", lambda event: submit_action())
submit_button.bind("<Button-3>", clock_out_all_users)  # Right-click event binding

# Visual feedback label
success_label = tk.Label(window, text="Entry was successful", font=("Arial", 18), bg=dark_background, fg="green")
success_label.place_forget()  # Hide initially

# Label to show the number of users currently clocked in
clocked_in_label = tk.Label(window, text="", font=("Arial", 14), bg=dark_background, fg=light_text)
clocked_in_label.place(relx=0.5, rely=0.7, anchor="center")

# Update clocked-in count on startup
update_clocked_in_count()

# Focus on the entry field on startup
entry.focus_set()

appName.place(relx=0.505, rely=0.2, anchor="n")
label.place(relx=0.5, rely=0.4, anchor="center")
entry.place(relx=0.5, rely=0.5, anchor="center")
submit_button.place(relx=0.5, rely=0.6, anchor="center")

bottom_buttons_frame = tk.Frame(window, bg=dark_background)
manual_entry_button = ttk.Button(bottom_buttons_frame, text="Manual Entry", command=manual_entry_action)
add_user_button = ttk.Button(bottom_buttons_frame, text="Add User", command=add_user_action)
review_button = ttk.Button(bottom_buttons_frame, text="Review", command=review_action)
manual_entry_button.pack(side="top", padx=5, pady=10)
add_user_button.pack(side="top", padx=5, pady=10)
review_button.pack(side="top", padx=5, pady=10)
bottom_buttons_frame.place(relx=0.5, rely=1.0, anchor="s")

close_button = ttk.Button(window, text="Close", command=close_application)
close_button.place(relx=1.0, rely=0.0, anchor="ne", x=-10, y=10)

window.mainloop()
