# libzim Checksum Verification Code (fileimpl.cpp)

- **Source URL**: https://raw.githubusercontent.com/openzim/libzim/main/src/fileimpl.cpp
- **Retrieved**: 2026-05-14

## FileImpl::verify() Method

```cpp
bool FileImpl::verify()
{
  if (!header.hasChecksum())
    return false;

  struct zim_MD5_CTX md5ctx;
  zim_MD5Init(&md5ctx);

  unsigned char ch[CHUNK_SIZE];
  offset_type checksumPos = header.getChecksumPos();
  offset_type toRead = checksumPos;

  for(auto part = zimFile->begin();
      part != zimFile->end();
      part++) {
    std::ifstream stream(part->second->filename(), 
                         std::ios_base::in|std::ios_base::binary);

    while(toRead>=CHUNK_SIZE && 
          stream.read(reinterpret_cast<char*>(ch),CHUNK_SIZE).good()) {
      zim_MD5Update(&md5ctx, ch, CHUNK_SIZE);
      toRead-=CHUNK_SIZE;
    }
    
    if(stream.good()){
      stream.read(reinterpret_cast<char*>(ch),toRead);
    }

    zim_MD5Update(&md5ctx, ch, stream.gcount());
    toRead-=stream.gcount();
  }

  unsigned char chksumCalc[16];
  auto chksumFile = zimReader->get_buffer(offset_t(header.getChecksumPos()), 
                                          zsize_t(16));

  zim_MD5Final(chksumCalc, &md5ctx);
  if (std::memcmp(chksumFile.data(), chksumCalc, 16) != 0)
  {
    return false;
  }
  return true;
}
```

## How Verification Works

1. Checks if header has a checksum field (hasChecksum())
2. Reads all file data from beginning up to checksumPos (NOT including the checksum itself)
3. Supports multi-part ZIM files (iterates over parts)
4. Reads in CHUNK_SIZE blocks (larger than the 1024 in creator.cpp)
5. Computes MD5 over all data bytes [0, checksumPos)
6. Reads the stored 16-byte checksum from the file at checksumPos
7. Compares computed vs stored — returns true if they match

## Key Observations

- Verification reads up to checksumPos only (excludes the 16-byte checksum itself)
- This matches creation: checksum covers [0, checksumPos), stored at [checksumPos, checksumPos+16)
- Multi-part file support means ZIM files can be split across multiple physical files
- The CHECKSUM entry in IntegrityCheck enum maps to this verify() method
- This is what `zimcheck` and kiwix-android's integrity checker call
