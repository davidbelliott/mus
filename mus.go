package main

import (
    "fmt"
    "os"
    //"os/exec"
    "os/user"
    "path/filepath"
    "bufio"
    "strings"
    "math/rand"
    "container/list"
)

// #include "fs.h"
// #cgo LDFLAGS: -lfluidsynth
import "C"

const order_fname = "order"
const music_cmd = "fluidsynth"
const music_args = "-a alsa -m alsa_seq -l -i /usr/share/soundfonts/OPL-3_FM_128M.sf2"

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

type Settings struct {
    autoplay bool
    quit bool
    paused bool
}

func (a Album) get_name() string {
    return a.name
}

func (a Album) get_filepaths(root_dir string) []string {
    filenames := make([]string, len(a.track_names))
    for i, fname := range a.track_names {
        filenames[i] = root_dir + "/" + a.name + "/" + fname
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
    return []string{root_dir + "/" + t.name}
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
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if info != nil && !info.IsDir() {
            files = append(files, path)
        }
        return nil
    })
    if err != nil {
        panic(err)
    }

    in_album := map[string]string{}

    n_root_tokens := len(strings.Split(root, "/"))
    for _, file := range files {
        tokens := strings.Split(file, "/")
        if len(tokens) >= n_root_tokens + 2 && tokens[len(tokens) - 1] == order_fname {
            album_name := strings.Join(tokens[n_root_tokens:len(tokens) - 1], "/")
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
                in_album[root + "/" + album_name + "/" + filename] = album_name
            }

            playables[album_name] = Album{name: album_name, track_names: track_names}
        }
    }

    for _, file := range files {
        _, ok := in_album[file]
        if !ok {
            tokens := strings.Split(file, "/")
            track_name := strings.Join(tokens[n_root_tokens:], "/")
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
        notify_str = fmt.Sprintf("%s > %s (%d/%d)", p.get_name(), fnames[i], i + 1, len(fnames))
    } else {
        notify_str = fmt.Sprintf("%s", p.get_name())
    }
    notify(notify_str)
}

func process_input(input string, queue *list.List, settings *Settings, playables *map[string]Playable) {
    input_tokens := strings.Split(input, " ")
    if len(input_tokens) == 1 && input_tokens[0] == "n" {
        C.stop()
    } else if len(input_tokens) == 1 && input_tokens[0] == "a" {
        settings.autoplay = !settings.autoplay
    } else if len(input_tokens) == 1 && input_tokens[0] == "q" {
        settings.quit = true
    } else if len(input_tokens) == 2 && input_tokens[0] == "p" {
        _, ok := (*playables)[input_tokens[1]]
        if ok {
            queue.PushBack(input_tokens[1])
        } else {
            notify("Track or album doesn't exist")
        }
    } else if len(input_tokens) == 1 && input_tokens[0] == "p" {
        if settings.paused {
            play()
        } else {
            pause()
        }
        settings.paused = !settings.paused
    }
}

func handle_input(input string, ok bool, queue *list.List, settings *Settings, playables *map[string]Playable) {
    if ok {
        process_input(input, queue, settings, playables)
        print_prompt()
    } else {
        settings.quit = true
    }
}

func play() {
    C.play()
    go wait_play_done(done_ch)
}

func pause() {
    C.pause()
    <-done_ch
}

func print_prompt() {
    //fmt.Print("> ")
}

// Channels
var input_ch = make(chan string)
var done_ch = make(chan bool)

func main() {
    C.init()
    user, err := user.Current()
    if err != nil {
        panic(err)
    }
    root := user.HomeDir + "/music/midi"

    playables := load_playables(root)

    if len(playables) == 0 {
        return
    }

    playable_names := make([]string, 0, len(playables))
    for k := range playables {
        playable_names = append(playable_names, k)
    }

    go read_input(input_ch)

    queue := list.New()
    settings := Settings{autoplay: true, quit: false, paused: false}

    print_prompt()
    for !settings.quit {
        if queue.Back() == nil {
            if settings.autoplay {
                queue.PushBack(playable_names[rand.Intn(len(playables))])
            } else {
                notify("No more tracks in queue")
                print_prompt()
                input, ok := <-input_ch
                handle_input(input, ok, queue, &settings, &playables)
            }
        } else {
            p_name := queue.Front()
            queue.Remove(p_name)
            p, ok := playables[p_name.Value.(string)]
            if !ok {
                continue
            }
            fpaths := p.get_filepaths(root)

            for i := 0; i < len(fpaths); i++ {

                C.add_midi(C.CString(fpaths[i]))
                play()

                notify_track(p, i)
                proceed := false

                for !proceed {
                    if settings.paused {
                        input, ok := <-input_ch
                        handle_input(input, ok, queue, &settings, &playables)
                    } else {
                        select {
                            case <-done_ch:
                                proceed = true
                            case input, ok := <-input_ch:
                                handle_input(input, ok, queue, &settings, &playables)
                        }
                    }
                }
            }
        }
    }
    C.cleanup()
}
