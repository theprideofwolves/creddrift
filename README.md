# CredDrift

CredDrift is a security tool built to find and track secrets in your code. It helps security teams manage risks by watching how secrets move and age. This project is designed for Governance Risk and Compliance (GRC) professionals who need to see the big picture of their security state.

## Core GRC Metrics

This tool does more than just find keys. It provides deep data on your risk:

* Blast Radius: Shows how many different files contain the same secret hash. A high number means a bigger risk.
* Drift Detection: Flags secrets that have moved into new folders or projects.
* Entropy Scoring: Uses math to find random strings that look like real keys.
* Rotation Tracking: Keeps a log of when you changed your keys to prove you follow security policies.



## Features

* Bulk Actions: Use checkboxes to manage many secrets at once. You can ignore or rotate a whole list with one click.
* Cross Platform Support: Works natively on Windows Mac and Linux.
* Native File Picker: Includes a Browse button that opens your actual system folder window.
* Automatic Cleanup: Marks secrets as Removed if they disappear from your files for more than 5 minutes.
* Glassmorphism UI: A modern dark mode dashboard that is easy to read.



## Installation

1. Ensure you have Go installed on your computer.
2. Clone this repository to your local machine.
3. Open your terminal in the project folder.
4. Run the command: go run main.go

## How to Use

1. Open your web browser and go to http://localhost:8080.
2. Click the Browse button to select your workspace or project folder.
3. Click Set Scan Path to start the scanner.
4. Review the secrets found in your code.
5. Select secrets using the checkboxes and use the Toolbar at the top to take action.

## Tech Stack

* Language: Go (Golang)
* Database: SQLite
* Frontend: HTML5 CSS3 and JavaScript
* Design: Custom Glassmorphism UI