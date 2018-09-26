package lrmp

import "time"

func byteToInt(buff []byte, offset int) int {
	var i uint

	i = uint(buff[offset]) << 24
	i |= uint(buff[offset+1]) << 16
	i |= uint(buff[offset+2]) << 8
	i |= uint(buff[offset+3])

	return int(i)
}

func byteToShort(buff []byte, offset int) int {

	return int(uint(buff[offset])<<8 | uint(buff[offset+1]))
}

func intToByte(i int, buff []byte, offset int) {
	buff[offset] = byte(i >> 24)
	buff[offset+1] = byte(i >> 16)
	buff[offset+2] = byte(i >> 8)
	buff[offset+3] = byte(i)
}

/**
 * converts milliseconds to 32 bit fixed point integer.
 * @param millis the milliseconds.
 */
func millisToFixedPoint32(millis int) int {
	/*
	 * expressed in units of 1/65536 seconds (1/0x10000).
	 * t32 = millis*2^16/1000
	 * use the factorization 2^16 = 2^6 + 2 - 58/125 which gives the exact value
	 * if no bit round error.
	 */
	return (millis << 6) + (millis << 1) - millis*58/125
}

const NtpOffsetSeconds int64 = 2208988800
const NtpOffsetMillis int64 = NtpOffsetSeconds * 1000
const NtpOffsetMillis32 = int(NtpOffsetSeconds << 16)

/**
 * converts UNIX time to 32 bit NTP time, i.e., 32-bit fixed point integer
 * (with fraction point at bit 16). The low 16 bits are the fraction part in
 * 1/2^16 second units.
 * @param millis the UNIX time.
 */
func ntp32(millis int64) int {
	millis += NtpOffsetMillis
	return (int)((millis << 16) / 1000)
}

/**
 * converts a 32 bit fixed point integer to milliseconds.
 * @param fixed the 32 bit fixed point integer.
 */
func fixedPoint32ToMillis(fixed int) int {

	/* fixed*1000/2^16 */

	fixed -= (fixed >> 7) * 3

	return (fixed + (1 << 5)) >> 6
}

func millis(d time.Duration) int64 {
	return d.Nanoseconds() / int64(time.Millisecond)
}

func addMillis(t time.Time, ms int) time.Time {
	return t.Add(time.Duration(ms) * time.Millisecond)
}
