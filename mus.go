package main

import (
    "fmt"
    "os"
    "os/user"
    "path/filepath"
    "bufio"
    "strings"
    "math/rand"
    "container/list"
    "errors"
    "flag"
    "log"
)

// #include "fs.h"
// #cgo LDFLAGS: -lfluidsynth
import "C"

const order_fname = "order"

const notify_cmd = "notify-send"

var input_file = os.Stdin

type Playable interface {
    get_name() string
    get_filepaths(root_dir string) []string
    get_filenames() []string
}

type Album struct {
    name string
    track_names []string
}

type Track struct {
    name string
}

type State struct {
    root string
    autoplay bool
    paused bool
    queue *list.List
    cur_playable Playable
    cur_idx int
}

func (a Album) get_name() string {
    return a.name
}

func (a Album) get_filepaths(root_dir string) []string {
    filenames := make([]string, len(a.track_names))
    for i, fname := range a.track_names {
        filenames[i] = filepath.Join(root_dir, a.name, fname)
    }
    return filenames
}

func (a Album) get_filenames() []string {
    return a.track_names
}

func (t Track) get_name() string {
    return t.name
}

func (t Track) get_filepaths(root_dir string) []string {
    return []string{filepath.Join(root_dir, t.name)}
}

func (t Track) get_filenames() []string {
    return []string{t.name}
}

func read_input(ch chan string) {
    reader := bufio.NewReader(input_file)
    for {
        s, err := reader.ReadString('\n')
        if err != nil {
            close(ch)
            return
        }
        ch <- s[:len(s) - 1]
    }
}

func wait_play_done(done_ch chan bool) {
    // Get the track to play
    C.wait()
    done_ch <- true
    return
}

func load_playables(root string) map[string]Playable {
    playables := map[string]Playable{}
    var files []string
    err := filepath.Walk(root, func(path string, info os.FileInfo,
        err error) error {
        if info != nil && !info.IsDir() {
            files = append(files, path)
        }
        return nil
    })
    if err != nil {
        panic(err)
    }

    in_album := map[string]string{}

    for _, file := range files {
        path, name := filepath.Split(file)
        if name == order_fname {
            album_name, _ := filepath.Rel(root, path)
            f, err := os.Open(file)
            if err != nil {
                panic(err)
            }
            defer f.Close()

            scanner := bufio.NewScanner(f)
            track_names := []string{}
            for scanner.Scan() {
                filename := scanner.Text()
                track_names = append(track_names, filename)
                in_album[filepath.Join(root, album_name, filename)] = album_name
            }

            playables[album_name] = Album{name: album_name,
                track_names: track_names}
        }
    }

    for _, file := range files {
        _, ok := in_album[file]
        if !ok {
            track_name, _ := filepath.Rel(root, file)
            playables[track_name] = Track{name: track_name}
        }
    }

    return playables
}

func notify(s string) {
    /*cmd := exec.Command(notify_cmd, s)
    err := cmd.Start()
    if err != nil {
        panic(err)
    }*/
    fmt.Println(s)
}

func notify_track(p Playable, i int) {
    fnames := p.get_filenames()
    var notify_str string
    if len(fnames) > 1 {
        notify_str = fmt.Sprintf("%s > %s (%d/%d)", p.get_name(), fnames[i],
            i + 1, len(fnames))
    } else {
        notify_str = fmt.Sprintf("%s", p.get_name())
    }
    notify(notify_str)
}

func enqueue_playable(name string, state *State) error {
    _, ok := playables[name]
    if !ok {
        return errors.New("nonexistent album/track")
    }
    state.queue.PushBack(name)
    return nil
}

func process_input(input string, ok bool, state *State,
    done_ch chan bool) bool {
    if !ok {
        return false
    }
    input_tokens := strings.Split(input, " ")
    if len(input_tokens) == 1 && input_tokens[0] == "n" {
        p, i := get_next_track(state)
        play_track(p, i, state, done_ch)
    } else if len(input_tokens) == 1 && input_tokens[0] == "a" {
        state.autoplay = !state.autoplay
    } else if len(input_tokens) == 1 && input_tokens[0] == "q" {
        return true
    } else if len(input_tokens) == 2 && input_tokens[0] == "p" {
        err := enqueue_playable(input_tokens[1], state)
        if err != nil {
            notify(err.Error())
        }
    } else if len(input_tokens) == 1 && input_tokens[0] == "p" {
        if state.paused {
            play(state, done_ch)
        } else {
            pause(state, done_ch)
        }
    }
    return false
}

func play(state *State, done_ch chan bool) {
    if (state.paused) {
        C.play()
        go wait_play_done(done_ch)
        state.paused = false
    }
}

func pause(state *State, done_ch chan bool) {
    if (!state.paused) {
        C.pause()
        <-done_ch
        state.paused = true
    }
}

func play_track(p Playable, i int, state *State, done_ch chan bool) {
    pause(state, done_ch)
    state.cur_playable = p
    state.cur_idx = i
    paths := p.get_filepaths(state.root)
    notify_track(p, i)
    if !C.add_midi(C.CString(paths[i])) {
        notify("couldn't add midi file")
    }
    play(state, done_ch)
}

func get_next_playable(state *State) Playable {
    if state.queue.Back() != nil {
        front := state.queue.Front()
        state.queue.Remove(front)
        return playables[front.Value.(string)]
    } else {
        return playables[playable_names[rand.Intn(len(playables))]]
    }
}

func get_next_track(state *State) (Playable, int) {
    var (
        p Playable
        i int
    )
    if state.cur_playable == nil {
        p = get_next_playable(state)
        i = 0
    } else {
        p = state.cur_playable
        i = state.cur_idx + 1
        if i >= len(state.cur_playable.get_filenames()) {
            p = get_next_playable(state)
            i = 0
        }
    }
    return p, i
}

const root_default_rel = "midi"

var playables map[string]Playable
var playable_names []string
var state State

var sound_driver = flag.String("driver", "pulseaudio",
    "the FluidSynth sound driver to use")
var soundfont = flag.String("soundfont", "/usr/share/soundfonts/default.sf2",
    "the FluidSynth soundfont to use")
var root string

func init() {
    user, err := user.Current()
    if err != nil {
        panic(err)
    }
    flag.StringVar(&root, "d", filepath.Join(user.HomeDir, root_default_rel),
        "the directory in which midi files are saved")

    flag.Parse()
    root, err = filepath.EvalSymlinks(root)
    if err != nil {
        log.Fatal(err)
    }

    C.init(C.CString(*sound_driver))
    C.load_soundfont(C.CString(*soundfont))
    state = State{root: root, autoplay: true, paused: true, queue: list.New(),
        cur_playable: nil, cur_idx: 0}

    playables = load_playables(state.root)
    if len(playables) == 0 {
        log.Fatal(fmt.Sprintf("no tracks or albums found in %s", state.root))
    }

    playable_names = make([]string, 0, len(playables))
    for k := range playables {
        playable_names = append(playable_names, k)
    }
}

func main() {
    var input_ch = make(chan string)
    go read_input(input_ch)

    var done_ch = make(chan bool)

    quit := false
    for !quit {
        if state.paused {
            input, ok := <-input_ch
            quit = process_input(input, ok, &state, done_ch)
        } else {
            select {
            case <-done_ch:
                state.paused = true
                p, i := get_next_track(&state)
                play_track(p, i, &state, done_ch)
            case input, ok := <-input_ch:
                quit = process_input(input, ok, &state, done_ch)
            }
        }
    }

    C.cleanup()
}
