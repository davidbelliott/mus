#ifndef FS_H
#define FS_H

#include <stdbool.h>

void init(const char* audio_driver);
void cleanup();
int load_soundfont(const char *fname);
bool add_midi(const char *fname);
void wait();
void play();
void pause();

#endif
