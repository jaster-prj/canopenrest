# canopenrest

`canopenrest` exposes a REST API for CANopen nodes. By default, it connects to
the Linux SocketCAN interface `can0`.

## Native simulator UDP-CAN

The service can use the native_sim UDP-CAN gateway instead of SocketCAN:

```sh
CAN_TRANSPORT=udp \
CAN_UDP_HOST=192.0.2.10 \
CAN_UDP_PORT=46100 \
go run ./cmd
```

`CAN_UDP_HOST` and `CAN_UDP_PORT` address the UDP-CAN gateway. Optionally set
`CAN_UDP_LISTEN` to choose the local UDP address; otherwise the OS selects an
ephemeral port. The transport uses the 72-byte little-endian frame format from
`zephyr-native-linux-udp-can` (CAN ID, DLC, flags, reserved bytes, and 64 data
bytes). The gateway must forward each request to the simulator and return its
response to the originating UDP endpoint for SDO operations to complete.

The transport is classic CAN only (DLC up to eight bytes). UDP does not
guarantee delivery, ordering, or retransmission; CANopen SDO timeouts still
apply.
