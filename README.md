# Hebbian Brain Simulation

A neuron-grid simulation written in Go, rendered with [Ebitengine](https://ebitengine.org/) (`ebiten/v2`).

## Local setup

### 1. Install Git

You need Git to clone the repository.

- **Windows:** download from https://git-scm.com/download/win and run the installer (defaults are fine).
- **macOS:** `xcode-select --install` (also installs the compiler tools Ebitengine needs).
- **Linux (Debian/Ubuntu):** `sudo apt install git`

Check it worked:

```sh
git --version
```

### 2. Install Go (1.23.4 or newer)

Download the installer for your OS from https://go.dev/dl/ and run it.

- **Windows:** run the `.msi` installer. It adds Go to your `PATH` automatically.
  You can also use winget:
  ```powershell
  winget install GoLang.Go
  ```
- **macOS:** run the `.pkg` installer, or use Homebrew:
  ```sh
  brew install go
  ```
- **Linux:** follow the tarball instructions at https://go.dev/doc/install, e.g.
  ```sh
  wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
  source ~/.profile
  ```

**Close and reopen your terminal**, then check it worked:

```sh
go version
```

You should see something like `go version go1.23.4 ...`.

### 3. Extra system packages (Linux only)

Ebitengine needs a C compiler and some graphics/audio headers on Linux. On Debian/Ubuntu:

```sh
sudo apt install gcc libc6-dev libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config
```

See https://ebitengine.org/en/documents/install.html for other distributions.
Windows and macOS need nothing extra.

### 4. Clone the repository

```sh
git clone https://github.com/AJLProjects/HebbianBrain.git
cd HebbianBrain
```

### 5. Move into the `src` directory

All Go code lives in `src`, and the program loads `map/brain-map.png` using a path relative to it, so it **must be run from inside `src`**:

```sh
cd src
```

### 6. Download dependencies

```sh
go mod download
```

(Optional — `go run` will also fetch these automatically the first time.)

### 7. Run it

```sh
go run .
```

The first run takes a little longer while Go downloads and compiles dependencies. A window with the simulation should open.

## Optional

### Build an executable

From inside `src`:

```sh
go build -o neuron-grid.exe .    # Windows
go build -o neuron-grid .        # macOS / Linux
```

Then run `./neuron-grid.exe` (or `./neuron-grid`) from the `src` directory.

### CPU profiling

```sh
go run . -cpuprofile cpu.prof -profileseconds 30
go tool pprof cpu.prof
```

- `-cpuprofile` — file to write the CPU profile to
- `-profileseconds` — how many seconds of the simulation to profile (default 30)

## Troubleshooting

| Problem | Fix |
| --- | --- |
| `go: command not found` / `'go' is not recognized` | Reopen your terminal after installing Go, or add Go's `bin` folder to your `PATH`. |
| `open map/brain-map.png: no such file or directory` | You're not in the `src` directory. Run `cd src` first. |
| `go.mod requires go >= 1.23.4` | Your Go is too old — install the latest from https://go.dev/dl/. |
| Linux build errors mentioning `GL`, `X11` or `alsa` | Install the packages in step 3. |
