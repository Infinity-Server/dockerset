/*
 *  Author: SpringHack - springhack@live.cn
 *  Last modified: 2022-01-18 23:49:22
 *  Filename: ffmpeg.c
 *  Description: Created by SpringHack using vim automatically.
 */
#include <unistd.h>

int main(int argc, char** argv) {
  setuid(0);
  setgid(0);
  return execvp("/usr/lib/jellyfin-ffmpeg/ffmpeg.exe", argv);
}
