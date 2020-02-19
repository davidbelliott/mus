#include <fluidsynth.h>
#include <stdbool.h>
#include "fs.h"

fluid_settings_t* settings;
fluid_synth_t* synth;
fluid_player_t* player;
fluid_audio_driver_t* adriver;

int load_soundfont(const char *fname);

void init(const char* audio_driver) {
    settings = new_fluid_settings();
    synth = new_fluid_synth(settings);
    fluid_settings_setstr(settings, "audio.driver", audio_driver);
    player = new_fluid_player(synth);
    fluid_player_stop(player);
    adriver = NULL;
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

// Must only be called when player is stopped and wait() has returned
bool add_midi(const char *fname) {
    if (fluid_player_get_status(player) != FLUID_PLAYER_DONE) {
        return false;
    }
    if (!fluid_is_midifile(fname)) {
        return false;
    }
    delete_fluid_player(player);
    player = new_fluid_player(synth);
    fluid_player_stop(player);
    fluid_player_add(player, fname);
    if (!adriver) {
        adriver = new_fluid_audio_driver(settings, synth);
    }
    return true;
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
