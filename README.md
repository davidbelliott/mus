# mus

Minimal MIDI player written in Go, using FluidSynth for synthesis.

## Basic usage

~~~
mus [-d dir] [-driver driver_name] [-soundfont soundfont]
~~~

`-d dir`: specifies the root directory to play MIDI music from. Default is `$HOME/midi`.

`-driver driver_name`: `driver_name` will be passed to FluidSynth, and can be any driver name supported by FluidSynth. Default is `pulseaudio`.

`-soundfont soundfont`: `soundfont` is the path to a FluidSynth soundfont file (.sf2) to use. Default is `/usr/share/soundfonts/default.sf2`.

`mus` will start paused with an empty queue. Singles and albums, inferred from the midi directory structure, can be enqueued using the `p trackname` command. Use the `p` command to play and pause, and the `n` command to skip. If `mus` is playing and no tracks are left on the queue, it will randomly select a single or album to play.

## MIDI directory structure

The root MIDI directory can contain any number of regular files and nested subdirectories. Upon initialization, files will be processed as follows:

1. Any subdirectory with a file named `order` will be classified as an album. The file `order` should contain a newline-separated list of filenames within the subdirectory in the order that they should be played. The album name will be the relative path of the subdirectory from the root directory.
2. Any regular file not belonging to an album will be classified as a single. The single name will be the relative path of the file from the root directory, including the file extension.

For example, consider the directory structure

~~~
midi
|   track1.mid
|
+---album1
|   |   order
|   |   track2.mid
|   |   track3.mid
|   
+---directory
    |   track4.mid
~~~

where the root directory is `midi` and the contents of the file `midi/album1/order` are

~~~
track3.mid
track2.mid
~~~

Then, the result will be one album (`album1`) with the order `track3.mid, track2.mid` and two singles: `track1.mid` and `directory/track4.mid`. If `album1` were a subdirectory within `directory`, it would then be named `directory/album1`.

## Interactive commands

`p`: play/pause

`n`: next track

`p [single|album]`: enqueue the specified single or album. The name of a single or album is determined by its relative path within the root MIDI directory, as detailed above.

`q`: quit

## Playlists

A playlist can be implemented as a file containing commands which is piped into `mus`, which immediately executes the commands in sequence. For instance, consider the file `playlist.txt` with contents:

~~~
p bach/organ/trio3c.mid
p bach/wtcbki
p
~~~

This playlist will add a single (`bach/organ/trio3c.mid`) and an album (`bach/wtcbki`), then play. It can be used as follows:

~~~
mus < playlist.txt          # runs all commands in playlist.txt
cat playlist.txt - | mus    # same, but takes input from stdin after
~~~
