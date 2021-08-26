# notacanserver

This partially implements the [panda v2 protocol](https://github.com/joshwardell/CANserver/wiki/PandaProtocol) for use with a generic Socketcan device, such as a Raspberry Pi with a [fancy hat](https://www.waveshare.com/2-ch-can-hat.htm). I probably won't answer questions but feel free to fork this if you find it useful.

It supports basic filtering, although removal of filters is not implemented, but all filters are cleared when a client disconnects.

To run on a host with one interface, just use:

```
./notacanserver can0
```

If you have multiple interfaces, they can all be handled by a single process, just append them to the command line:

```
./notacanserver can0 can1
```
