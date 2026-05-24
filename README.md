# Rummix

Rummix is a cross-platform, tile-based Rummy game (inspired by the classic Rummikub rules) developed in Go using the [Fyne](https://fyne.io/) toolkit.

## Features

*   **Intuitive GUI**: Drag-and-drop tiles to organize your rack and create combinations on the game board.
*   **AI Opponents**: Play against 1 to 3 challenging AI players.
*   **Customizable Rules**:
    *   Configurable opening points (25, 30, or 50).
    *   Optional turn time limits (1-5 minutes).
*   **Multilingual Support**: Fully localized in English and French.
*   **Theming**: Support for both Light and Dark modes.
*   **Statistics**: Tracks wins, games played, and scores for all players.
*   **Sound Effects**: Audio feedback for valid/invalid moves and game events.

## Prerequisites

*   **Go**: Version 1.18 or higher recommended.
*   **Fyne Dependencies**: Ensure your system has the necessary drivers for Fyne (C compiler and graphics development headers). See [Fyne's getting started guide](https://developer.fyne.io/started/) for details.

## Installation & Running

Pre-compiled binaries are available in the [Releases](https://github.com/jplozf/rummix/releases) tab of this repository. Alternatively, you can build and run from source following these steps:

1.  Clone the repository:
    ```bash
    git clone https://github.com/jplozf/rummix.git
    cd rummix
    ```

2.  Install dependencies:
    ```bash
    go mod tidy
    ```

3.  Run the application:
    ```bash
    go run .
    ```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

© JPL 2026