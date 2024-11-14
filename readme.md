# Bailey Butler Lock Screen

This program replaces your wallpaper and lock screen with the current state (within 10 minutes) of the website [isbaileybutlerintheoffice.today](https://isbaileybutlerintheoffice.today), which tells us if Bailey Butler is in the office. It is recommended to use this in conjunction with applications like Caffeine or Amphetamine to keep your computer awake when locked so passersby can also see if Bailey is in the office.

## Features

- Automatically updates your wallpaper and lock screen every 10 minutes (configurable).
- Uses headless Chrome to capture a screenshot of the website.
- Retries up to 3 times in case of errors.
- Logs detailed information for troubleshooting.
- Configurable screenshot/window size.

## Requirements

- Go 1.23.2 or later
- macOS (for setting the lock screen and wallpaper)
- [Caffeine](https://lightheadsw.com/caffeine/) or [Amphetamine](https://apps.apple.com/us/app/amphetamine/id937984704?mt=12) (optional, to keep the computer awake)

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/jamesblewis/baileybutlerlockscreen.git
    cd baileybutlerlockscreen
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

3. Build the project:
    ```sh
    go build -o baileybutlerlockscreen cmd/main.go
    ```

## Usage

Run the program:
```sh
make run
