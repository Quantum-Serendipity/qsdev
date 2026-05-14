# libzim Checksum Creation Code (creator.cpp)

- **Source URL**: https://raw.githubusercontent.com/openzim/libzim/main/src/writer/creator.cpp
- **Retrieved**: 2026-05-14

## Checksum Computation in writeLastParts()

```cpp
TINFO(" write checksum");
struct zim_MD5_CTX md5ctx;
unsigned char batch_read[1024+1];
lseek(out_fd, 0, SEEK_SET);
zim_MD5Init(&md5ctx);
while (true) {
   auto r = read(out_fd, batch_read, 1024);
   if (r == -1) {
     throw std::runtime_error(std::strerror(errno));
   }
   if (r == 0)
     break;
   batch_read[r] = 0;
   zim_MD5Update(&md5ctx, batch_read, r);
}
unsigned char digest[16];
zim_MD5Final(digest, &md5ctx);
_write(out_fd, reinterpret_cast<const char*>(digest), 16);
```

## Process

1. After all content, pointers, and the header have been written to the file
2. Seeks back to position 0
3. Reads the entire file in 1024-byte chunks
4. Feeds each chunk into MD5 context
5. Finalizes the MD5 digest (16 bytes)
6. Appends the 16-byte digest to the end of the file

## Key Observations

- The checksum covers bytes [0, EOF) — the entire file as written before the checksum
- The checksum is the last 16 bytes of the final file
- The header field `checksumPos` points to the offset where this checksum starts
- MD5 is computed AFTER all data is written — this is the "integrity gap" described in issue #614
- Uses a custom `zim_MD5` implementation (not OpenSSL)
- 1024-byte read buffer is small for large files (contributors noted this is inefficient)
