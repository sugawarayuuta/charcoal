# Faster utf8.Valid using multi-byte processing without SIMD.

## Introduction

Hello! I'm Sugawara Yuuta. I wrote about my fast floating point parser in the previous article. And I wondered if I could improve the reading part faster, which led to this experiment. This is about reading multiple bytes at the same time, without SIMD or any special instructions like that. Let's Go!

## Today's goal, non-goal

We are going to try improving the performance of Go's standard library, `utf8.Valid`. This function validates the encoding (UTF-8) of given byte slices. Of course, there are people that studied on this topic in the past, including Daniel Lemire's paper. Unlike the paper, I'm limiting myself from using SIMD.

The reason? Platform compatibility. It really simplifies the code for multiple platforms, and maintenance. Especially in Go, the assembly is a bit hard to use (at least for me), `uint64` is special in a sense that it's available and supported on all platforms Go supports, including 32-bit ones.

So, let me talk about UTF-8 for a bit.

## UTF-8

It's an encoding, and some of the people in the Go team made it! Specifically, it builds a byte data of integers that have characters mapped to it. It sounds easy, until you learn how it works. For example, the encoding that was used a lot in Japan, Shift-JIS, has overlaps in bytes, so simply searching for byte matches in it gives us false positives.

Other than that, UTF-8 has a perfect compatibility with ASCII, things like `a`, `A`, `0`, and it makes me appreciate the encoding even more.

So back to the original topic. the Problem is to validate data so that we know for sure it follows the rules. You may ask if we actually need that. Well, there used to be some vulnerabilities that were caused by poor UTF-8 validation and handling, so the answer seems to be pretty clear, we need it.

But how do we do it?

## Validating multiple bytes at once

Normally, computers have 64-bit words, meaing, we can calculate multiple bytes at once, up to 8 bytes (1 byte = 8-bit).

Fortunately, Go has optimized memory combination (it just combines multiple bytes to make one big word, like [this](https://github.com/golang/go/blob/c2c4a32f9e57ac9f7102deeba8273bcd2b205d3c/src/cmd/compile/internal/ssa/memcombine.go)), so this wouldn't be a problem.

The actual calculation goes like this.

### Detecting ASCII

As I said, ASCII characters are valid part of UTF-8. And, alphanumerics do exist often, so it's important to make a fast-path of ASCII detection.

It's not difficult! At all! ASCIIs

- are always represented as a single byte.

- have the MSB (most significant bit) 0.

To get only the MSB, we use bit-wise AND (`&`). The mask is `0x8080808080808080` in hexadecimal, because it has `0b10000000` mapped into each byte.

An implementation of it can be something like...

```go
func isASCII(data uint64) bool {
	return data&0x8080808080808080 == 0
}
```

([go.dev/play](https://go.dev/play/p/bl3CAjP2Z8c))

### Detecting always-invalid bytes

What do we do when we encounter bytes that are not ASCII? Firstly I looked at the bytes that don't exist anywhere in valid UTF-8. Specifically,

- bytes that are bigger than `0xf4` don't exist in valid UTF-8.

- bytes that are `0xc0` or `0xc1` don't exist in valid UTF-8.

So we can make the problem easier, now it's a simple byte matching. An implementation of this can be

```go
func contains(data, mask uint64) bool {
	data ^= mask
	return (data-0x0101010101010101)&^data&0x8080808080808080 != 0
}
```

([go.dev/play](https://go.dev/play/p/xhbhc4GVzo1))

`mask` has the character you want to search in its each byte slot. For the playground link above, it's searching for `'0'` == `0x30`, making the mask `0x3030303030303030`. The `^` is XOR, not NOT in this case. And it makes the byte 0 when it matches, which overflows when we subtract 1 from it.

Also `&^` is bit-wise AND NOT. It excludes the cases where the MSB was 1 from the start, to prevent false positives.

To search 2 characters that are next to each other, you can use bit-wise OR to make less instructions.

Similarly, you can search for bytes that are bigger than `0xf4` by overflowing matched bytes.

### Detecting conditionally-invalid bytes

Reading the [Unicode Standard](https://www.unicode.org/versions/Unicode15.0.0/UnicodeStandard-15.0.pdf#G6.27506), sometimes second bytes have a range that is more limited than others. Specifically `0xe0`, `0xed`, `0xf0`, `0xf4`. The default cases are handled later.

Using the trick we just learned...

```go
func isSpecial(data uint64) bool {
	xed := data ^ 0xedededededededed
	xf0 := data | 0x1010101010101010 ^ 0xf0f0f0f0f0f0f0f0
	xf4 := data ^ 0xf4f4f4f4f4f4f4f4
	xed = (xed - 0x0101010101010101) &^ xed
	xf0 = (xf0 - 0x0101010101010101) &^ xf0
	xf4 = (xf4 - 0x0101010101010101) &^ xf4
	return (xed|xf0|xf4)&0x8080808080808080 != 0
}
```

([go.dev/play](https://go.dev/play/p/FeOK65FQTd6))

But we can reduce the instructions further! Currently it used 16 instructions, counting AND-NOT as 2. So look at this.

```go
func isSpecial(data uint64) bool {
	top := data & 0x8080808080808080
	btm := data & 0x7f7f7f7f7f7f7f7f
	xed := btm ^ 0x6d6d6d6d6d6d6d6d - 0x0101010101010101
	xf0 := btm | 0x1010101010101010 ^ 0x7070707070707070 - 0x0101010101010101
	xf4 := btm ^ 0x7474747474747474 - 0x0101010101010101
	return top&(xed|xf0|xf4) != 0
}
```

([go.dev/play](https://go.dev/play/p/l4z4Rxz448S))

Now we have 12 instructions. The trick is to have the MSB bit and non-MSB bits separated, allowing us to prevent the false positives we discussed earlier in one step.

Just like that we can optimize things further when there are more patterns to our data.

### The order of bytes

Finally, we check the order. We need to leave the leading ones and delete the rest, but this is difficult. If the problem was trailing bits, not leading, however, it would be easy. Because the implementation becomes 1 line.

```go
data&(^data-0x0101010101010101)
```

This is not the case, though. so We need to reverse bits if we want to use this. Unfortunately this is not the fastest, An easy implementation will be like below but it's harder for CPUs (I guess?) to optimize this.

```go
func reverse64(data uint64) uint64 {
	data = data>>1&0x5555555555555555 | data&0x5555555555555555<<1
	data = data>>2&0x3333333333333333 | data&0x3333333333333333<<2
	data = data>>4&0x0f0f0f0f0f0f0f0f | data&0x0f0f0f0f0f0f0f0f<<4
	return data
}
```

([go.dev/play](https://go.dev/play/p/uJwJHGNxOWv))

Is there a approach that doesn't need a reversal? 
Yes. it's not hard either, with some steps...

```go
	u64 = ^u64
	u64 |= u64 & 0xfefefefefefefefe >> 1
	u64 |= u64 & 0xfcfcfcfcfcfcfcfc >> 2
	u64 |= u64 & 0xf0f0f0f0f0f0f0f0 >> 4
	u64 = ^u64
```

([go.dev/play](https://go.dev/play/p/ylU3MUbW4J1))

Great news: We can make this faster! We already excluded the bytes that has more than 4 leading one bits, in the "Detecting always-invalid bytes" part. So we don't actually need the 3rd OR line.

And, we can now `bits.Mul64` the first byte and match it to the other parts, if it does, then we finished validating. It's valid. If this multiplication overflows into MSBs, then do the check with next block.

OK, how does it perform?

## Benchmarks

The below chart shows how many bytes we can validate in a second (higher is better). The left side is the standard library and the right side is my library.

```
                     │ ./before.txt │              ./after.txt              │
                     │     B/s      │      B/s       vs base                │
Valid/ascii-small-4    1.699Gi ± 2%    1.783Gi ± 1%   +4.99% (p=0.000 n=10)
Valid/ascii-large-4    11.01Gi ± 1%    15.82Gi ± 1%  +43.64% (p=0.000 n=10)
Valid/kanji-small-4    982.4Mi ± 1%   1099.9Mi ± 2%  +11.95% (p=0.000 n=10)
Valid/kanji-large-4    890.4Mi ± 1%   1334.1Mi ± 1%  +49.82% (p=0.000 n=10)
Valid/unicode.json-4   821.6Mi ± 7%   1226.3Mi ± 2%  +49.26% (p=0.000 n=10)
geomean                1.658Gi         2.162Gi       +30.44%
```

Wow. that looks great.

`ascii-small` is 10 characters of ASCII, `ascii-large` is 100KB is ASCII. It improved anyway because I improved the bounds check (convincing the compiler is tricky; I spent a lot of time with it too)... Also it uses less small memory loads for any slices that are bigger than 8 bytes.

`kanji-small` is 10 characters of Japanese Kanji, and `kanji-large` is 100KB of them. Including `unicode.json`, which has many Unicode runes from many categories, It's improving because of the algorithm we discussed earlier.

## Thank you for reading

Thank you for reading. If you have questions, contributions, ideas, suggestions or anything like that, let me know!