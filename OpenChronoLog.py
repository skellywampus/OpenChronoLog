import tkinter as tk
from tkinter import ttk, simpledialog, messagebox, filedialog
from datetime import datetime, timedelta
import json
import os
import csv

USER_DB_FILE = "user_database.json"
SCAN_LOG_FILE = "scan_log.json"

# Load and Save Data Functions
def load_data():
    global user_database, scan_log
    user_database = []
    scan_log = []
    if os.path.exists(USER_DB_FILE):
        with open(USER_DB_FILE, "r") as f:
            user_database = json.load(f)
    if os.path.exists(SCAN_LOG_FILE):
        with open(SCAN_LOG_FILE, "r") as f:
            scan_log = json.load(f)

def save_data():
    with open(USER_DB_FILE, "w") as f:
        json.dump(user_database, f, indent=4)
    with open(SCAN_LOG_FILE, "w") as f:
        json.dump(scan_log, f, indent=4)

def find_user_by_id(user_id):
    return next((user for user in user_database if user["id"] == user_id), None)

def update_clocked_in_count():
    count = sum(user["status"] == "clocked_in" for user in user_database)
    clocked_in_label.config(text=f"{count} users currently clocked in")

def submit_action():
    user_id = entry.get().strip()
    if not user_id:
        messagebox.showerror("Error", "User ID cannot be empty")
        return
    user = find_user_by_id(user_id)
    if user:
        handle_scan(user)
        save_data()
        update_clocked_in_count()
        success_label.place(relx=0.5, rely=0.65, anchor="center")
        window.after(2000, success_label.place_forget)
    else:
        messagebox.showerror("Error", f"User with ID {user_id} not found")
    entry.delete(0, tk.END)
    entry.focus_set()

def handle_scan(user):
    current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    user["status"] = "clocked_out" if user["status"] == "clocked_in" else "clocked_in"
    if user["status"] == "clocked_out":
        calculate_duration(user["id"], current_time)
    scan_log.append({
        "user_id": user["id"],
        "name": user["name"],
        "time": current_time,
        "status": user["status"],
        "entry_type": "Scan"
    })

def calculate_duration(user_id, clock_out_time):
    clock_in_time = next(
        (entry["time"] for entry in reversed(scan_log)
         if entry["user_id"] == user_id and entry["status"] == "clocked_in"), None)
    if clock_in_time:
        clock_in_dt = datetime.strptime(clock_in_time, "%Y-%m-%d %H:%M:%S")
        clock_out_dt = datetime.strptime(clock_out_time, "%Y-%m-%d %H:%M:%S")
        duration = clock_out_dt - clock_in_dt
        print(f"{user_id} was clocked in for {duration}")

def manual_entry_action():
    user_id = simpledialog.askstring("Manual Entry", "Enter User ID:", parent=window)
    if not user_id:
        return
    time_in = simpledialog.askstring("Manual Entry", "Enter Time In (HH:MM:SS):", parent=window)
    if not time_in:
        return
    time_out = simpledialog.askstring("Manual Entry", "Enter Time Out (HH:MM:SS):", parent=window)
    if not time_out:
        return
    current_date = datetime.now().strftime('%Y-%m-%d')
    time_in_full = f"{current_date} {time_in}"
    time_out_full = f"{current_date} {time_out}"
    user = find_user_by_id(user_id)
    if user:
        user["status"] = "clocked_out"
        calculate_duration(user["id"], time_out_full)
        scan_log.extend([
            {"user_id": user["id"], "name": user["name"], "time": time_in_full, "status": "clocked_in", "entry_type": "Manual"},
            {"user_id": user["id"], "name": user["name"], "time": time_out_full, "status": "clocked_out", "entry_type": "Manual"}
        ])
        save_data()
    else:
        messagebox.showerror("Error", f"User with ID {user_id} not found", parent=window)

def add_user_action():
    user_id = simpledialog.askstring("Add User", "Enter User ID:", parent=window)
    if not user_id:
        return
    user_name = simpledialog.askstring("Add User", "Enter User Name:", parent=window)
    if not user_name:
        return
    if find_user_by_id(user_id):
        messagebox.showerror("Error", "User with this ID already exists", parent=window)
        return
    user_database.append({"id": user_id, "name": user_name, "status": "clocked_out"})
    save_data()

def review_action():
    review_window = tk.Toplevel(window)
    review_window.title("Review Data")

    def show_user_list():
        display_data("User List", user_database, ["id", "name"])

    def show_scan_log():
        log_window, tree = display_data("Scan Log", scan_log, ["user_id", "name", "time", "status", "entry_type"])

        def delete_entry():
            selected_item = tree.selection()
            if not selected_item:
                messagebox.showerror("Error", "No item selected")
                return
            confirmation = messagebox.askyesno("Confirm Delete", "Are you sure you want to delete the selected entry?")
            if confirmation:
                item = tree.item(selected_item)
                entry_values = item["values"]
                scan_log[:] = [entry for entry in scan_log if not all(
                    str(entry[col]) == str(val) for col, val in zip(["user_id", "name", "time", "status", "entry_type"], entry_values))]
                tree.delete(selected_item)
                save_data()
                messagebox.showinfo("Success", "Entry deleted successfully")

        def modify_entry():
            selected_item = tree.selection()
            if not selected_item:
                messagebox.showerror("Error", "No item selected")
                return
            item = tree.item(selected_item)
            entry_values = item["values"]
            new_time = simpledialog.askstring("Modify Entry", "Enter new time (YYYY-MM-DD HH:MM:SS):", initialvalue=entry_values[2])
            if new_time is None:
                return
            new_status = simpledialog.askstring("Modify Entry", "Enter new status (clocked_in/clocked_out):", initialvalue=entry_values[3])
            if new_status is None:
                return
            for entry in scan_log:
                if all(str(entry[col]) == str(val) for col, val in zip(["user_id", "name", "time", "status", "entry_type"], entry_values)):
                    entry["time"] = new_time
                    entry["status"] = new_status
                    break
            tree.item(selected_item, values=(entry_values[0], entry_values[1], new_time, new_status, entry_values[4]))
            save_data()
            messagebox.showinfo("Success", "Entry modified successfully")

        delete_button = ttk.Button(log_window, text="Delete Entry", command=delete_entry)
        delete_button.pack(pady=5)
        modify_button = ttk.Button(log_window, text="Modify Entry", command=modify_entry)
        modify_button.pack(pady=5)

    def show_time_data():
        time_data_window = tk.Toplevel(review_window)
        time_data_window.title("Time Data")
        tk.Label(time_data_window, text="Select Start Date:").grid(row=0, column=0)
        tk.Label(time_data_window, text="Select End Date:").grid(row=1, column=0)
        start_month = ttk.Combobox(time_data_window, values=[f"{i:02d}" for i in range(1, 13)], width=5)
        start_year = ttk.Combobox(time_data_window, values=[str(i) for i in range(2000, 2051)], width=5)
        end_month = ttk.Combobox(time_data_window, values=[f"{i:02d}" for i in range(1, 13)], width=5)
        end_year = ttk.Combobox(time_data_window, values=[str(i) for i in range(2000, 2051)], width=5)
        start_month.grid(row=0, column=1); start_year.grid(row=0, column=2)
        end_month.grid(row=1, column=1); end_year.grid(row=1, column=2)
        start_month.set("01"); start_year.set(str(datetime.now().year))
        end_month.set("01"); end_year.set(str(datetime.now().year))

        def show_today():
            end_month.set(datetime.now().strftime('%m'))
            end_year.set(datetime.now().strftime('%Y'))

        ttk.Button(time_data_window, text="Today", command=show_today).grid(row=1, column=3)

        def calculate_time_data():
            start_date = f"{start_year.get()}-{start_month.get()}-01"
            end_date = f"{end_year.get()}-{end_month.get()}-31"
            total_time = {}
            for entry in scan_log:
                if start_date <= entry["time"].split(" ")[0] <= end_date:
                    user_id = entry["user_id"]
                    total_time.setdefault(user_id, timedelta())
                    if entry["status"] == "clocked_in":
                        clock_in_time = datetime.strptime(entry["time"], "%Y-%m-%d %H:%M:%S")
                    elif entry["status"] == "clocked_out":
                        clock_out_time = datetime.strptime(entry["time"], "%Y-%m-%d %H:%M:%S")
                        total_time[user_id] += clock_out_time - clock_in_time
            time_data_list = [{"user_id": uid, "name": find_user_by_id(uid)["name"], "time_worked": str(total)} for uid, total in total_time.items()]
            display_data("Time Data", time_data_list, ["user_id", "name", "time_worked"])

        ttk.Button(time_data_window, text="Calculate", command=calculate_time_data).grid(row=2, column=0, columnspan=4, pady=10)

    def export_data():
        file_path = filedialog.asksaveasfilename(defaultextension=".csv")
        if file_path:
            try:
                with open(file_path, 'w', newline='') as file:
                    writer = csv.writer(file)
                    writer.writerow(["User ID", "Name", "Time", "Status", "Entry Type"])
                    for entry in scan_log:
                        writer.writerow([entry[col] for col in ["user_id", "name", "time", "status", "entry_type"]])
                messagebox.showinfo("Export Successful", f"Data exported to {file_path}")
            except Exception as e:
                messagebox.showerror("Export Failed", f"An error occurred: {e}")

    ttk.Button(review_window, text="User List", command=show_user_list).pack(pady=5)
    ttk.Button(review_window, text="Scan Log", command=show_scan_log).pack(pady=5)
    ttk.Button(review_window, text="Time Data", command=show_time_data).pack(pady=5)
    ttk.Button(review_window, text="Export Data", command=export_data).pack(pady=5)

def clock_out_all_users(event=None):
    if messagebox.askyesno("Confirm Clock Out", "Are you sure you want to clock out all users?"):
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
        messagebox.showinfo("All Users Clocked Out", "All users have been clocked out.")

def display_data(title, data, columns):
    data_window = tk.Toplevel(window)
    data_window.title(title)
    tree = ttk.Treeview(data_window, columns=columns, show='headings')
    for col in columns:
        tree.heading(col, text=col, command=lambda _col=col: sort_treeview_column(tree, _col, False))
        tree.column(col, minwidth=0, width=100, stretch=tk.NO)
    for item in data:
        tree.insert("", tk.END, values=[item[col] for col in columns])
    tree.pack(expand=True, fill=tk.BOTH)
    return data_window, tree

def sort_treeview_column(tree, col, reverse):
    l = [(tree.set(k, col), k) for k in tree.get_children('')]
    l.sort(reverse=reverse)
    for index, (_, k) in enumerate(l):
        tree.move(k, '', index)
    tree.heading(col, command=lambda: sort_treeview_column(tree, col, not reverse))

def view_clocked_in_users():
    clocked_in_users = [user for user in user_database if user["status"] == "clocked_in"]
    if not clocked_in_users:
        messagebox.showinfo("No Users Clocked In", "There are no users currently clocked in.")
        return
    window_ci = tk.Toplevel()
    window_ci.title("Users Currently Clocked In")
    columns = ["id", "name"]
    tree = ttk.Treeview(window_ci, columns=columns, show='headings')
    for col in columns:
        tree.heading(col, text=col)
        tree.column(col, minwidth=0, width=100, stretch=tk.NO)
    for user in clocked_in_users:
        tree.insert("", tk.END, values=(user["id"], user["name"]))
    tree.pack(expand=True, fill=tk.BOTH)

    def clock_out_selected_user():
        selected_item = tree.selection()
        if not selected_item:
            messagebox.showerror("Error", "No user selected")
            return
        item = tree.item(selected_item)
        user_id = item["values"][0]
        user = find_user_by_id(user_id)
        if user:
            current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            user["status"] = "clocked_out"
            calculate_duration(user_id, current_time)
            scan_log.append({
                "user_id": user["id"],
                "name": user["name"],
                "time": current_time,
                "status": "clocked_out",
                "entry_type": "Manual"
            })
            save_data()
            update_clocked_in_count()
            tree.delete(selected_item)
            messagebox.showinfo("Success", f"{user['name']} has been clocked out.")
        else:
            messagebox.showerror("Error", "User not found")

    clock_out_button = ttk.Button(window_ci, text="Clock Out Selected User", command=clock_out_selected_user)
    clock_out_button.pack(pady=10)

def close_application():
    if messagebox.askokcancel("Quit", "Do you want to quit?"):
        window.destroy()

# UI Setup
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
window.attributes("-fullscreen", True)
window.bind("<F11>", lambda e: window.attributes("-fullscreen", not window.attributes("-fullscreen")))
window.bind("<Escape>", lambda e: window.attributes("-fullscreen", False))

style = ttk.Style()
style.configure("TButton", background=button_color, foreground=button_text_color, font=("Arial", "12"))
style.map("TButton", background=[("active", button_hover)])

appName = tk.Label(window, text="OpenChronoLog", font=("Arial", 55), bg=dark_background, fg=light_text)
label = tk.Label(window, text="Enter user ID", font=("Arial", 28), bg=dark_background, fg=light_text)
entry = tk.Entry(window, width=30, bg=text_box_color, fg=light_text, insertbackground=light_text, relief=tk.FLAT)
submit_button = ttk.Button(window, text="Submit", command=submit_action)
entry.bind("<Return>", lambda event: submit_action())
submit_button.bind("<Button-3>", clock_out_all_users)
success_label = tk.Label(window, text="Entry was successful", font=("Arial", 18), bg=dark_background, fg="green")
success_label.place_forget()

# Clocked-in users label and View button
clocked_in_frame = tk.Frame(window, bg=dark_background)
clocked_in_label = tk.Label(clocked_in_frame, text="", font=("Arial", 14), bg=dark_background, fg=light_text)
view_button = ttk.Button(clocked_in_frame, text="View", command=view_clocked_in_users)
clocked_in_label.pack(side="left")
view_button.pack(side="left", padx=10)
clocked_in_frame.place(relx=0.5, rely=0.7, anchor="center")

update_clocked_in_count()
entry.focus_set()
appName.place(relx=0.5, rely=0.2, anchor="n")
label.place(relx=0.5, rely=0.4, anchor="center")
entry.place(relx=0.5, rely=0.5, anchor="center")
submit_button.place(relx=0.5, rely=0.6, anchor="center")

bottom_buttons_frame = tk.Frame(window, bg=dark_background)
manual_entry_button = ttk.Button(bottom_buttons_frame, text="Manual Entry", command=manual_entry_action)
add_user_button = ttk.Button(bottom_buttons_frame, text="Add User", command=add_user_action)
review_button = ttk.Button(bottom_buttons_frame, text="Review", command=review_action)
manual_entry_button.pack(side="top", padx=5, pady=5)
add_user_button.pack(side="top", padx=5, pady=5)
review_button.pack(side="top", padx=5, pady=5)
bottom_buttons_frame.place(relx=0.5, rely=1.0, anchor="s")

close_button = ttk.Button(window, text="Close", command=close_application)
close_button.place(relx=1.0, rely=0.0, anchor="ne", x=-10, y=10)

window.mainloop()
