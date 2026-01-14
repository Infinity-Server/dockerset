#define _GNU_SOURCE
#include <X11/Xatom.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <dlfcn.h>
#include <stdio.h>
#include <string.h>

Status XIconifyWindow(Display* display, Window w, int screen_number) {
  return 1;
}

Status XSendEvent(Display* display, Window w, Bool propagate, long event_mask,
                  XEvent* event_send) {
  static Status (*orig_XSendEvent)(Display*, Window, Bool, long, XEvent*) =
      NULL;
  if (!orig_XSendEvent) orig_XSendEvent = dlsym(RTLD_NEXT, "XSendEvent");

  if (event_send->type == ClientMessage) {
    Atom msg_type = event_send->xclient.message_type;
    Atom wm_state = XInternAtom(display, "_NET_WM_STATE", False);

    if (msg_type == wm_state) {
      long action = event_send->xclient.data.l[0];
      Atom prop = (Atom)event_send->xclient.data.l[1];
      Atom hidden = XInternAtom(display, "_NET_WM_STATE_HIDDEN", False);

      if (prop == hidden) {
        return 1;
      }
    }
  }
  return orig_XSendEvent(display, w, propagate, event_mask, event_send);
}

int XChangeProperty(Display* display, Window w, Atom property, Atom type,
                    int format, int mode, const unsigned char* data,
                    int nelements) {
  static int (*orig_XChangeProperty)(Display*, Window, Atom, Atom, int, int,
                                     const unsigned char*, int) = NULL;
  if (!orig_XChangeProperty)
    orig_XChangeProperty = dlsym(RTLD_NEXT, "XChangeProperty");

  Atom wm_hints = XInternAtom(display, "WM_HINTS", False);
  if (property == wm_hints && data) {
    XWMHints* hints = (XWMHints*)data;
    if (hints->flags & StateHint && hints->initial_state == IconicState) {
      hints->initial_state = NormalState;
    }
  }
  return orig_XChangeProperty(display, w, property, type, format, mode, data,
                              nelements);
}
