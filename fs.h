#ifndef FS_H
#define FS_H

void init();
void cleanup();
int load_soundfont(const char *fname);
int add_midi(const char *fname);
void wait();
void play();
void pause();
void stop();

#endif
