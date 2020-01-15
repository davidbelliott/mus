#include <fluidsynth.h>

fluid_settings_t* settings;
fluid_synth_t* synth;
fluid_player_t* player;
fluid_audio_driver_t* adriver;

int load_soundfont(const char *fname);

void init() {
    settings = new_fluid_settings();
    synth = new_fluid_synth(settings);
    fluid_settings_setstr(settings, "audio.driver", "alsa");
    player = new_fluid_player(synth);
    adriver = NULL;
    load_soundfont("/usr/share/soundfonts/default.sf2");
}

void cleanup() {
    if (adriver) {
        delete_fluid_audio_driver(adriver);
    }
    delete_fluid_player(player);
    delete_fluid_synth(synth);
    delete_fluid_settings(settings);
}

int load_soundfont(const char *fname) {
    if (fluid_is_soundfont(fname)) {
        fluid_synth_sfload(synth, fname, 1);
        return 0;
    }
    return 1;
}

int add_midi(const char *fname) {
    if (!fluid_is_midifile(fname)) {
        return 1;
    }
    int ticks = fluid_player_get_total_ticks(player);
    fluid_player_add(player, fname);
    if (!adriver) {
        adriver = new_fluid_audio_driver(settings, synth);
    }
    fluid_player_seek(player, ticks);
    return 0;
}

void wait() {
    fluid_player_join(player);
}

void play() {
    fluid_player_play(player);
}

void pause() {
    fluid_player_stop(player);
}

void stop() {
    fluid_player_stop(player);
}
