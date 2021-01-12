This is the file struture of the `accounts/` files.

Extrapolated from source in:
* runtime/src/append_vec.rs
* runtime/src/accounts_db.rs
* runtime/src/hash.rs



pub struct StoredMeta {
    pub write_version: u64,
    pub pubkey: Pubkey,      FOUND
    pub data_len: u64,       FOUND
}
pub struct AccountMeta {
    pub lamports: u64,
    pub owner: Pubkey,       FOUND
    pub executable: bool,
    pub rent_epoch: Epoch,   FOUND
}


STRUCTURE:
                (meta_ptr as *const u8, mem::size_of::<StoredMeta>()),              PARTIALLY
                (account_meta_ptr as *const u8, mem::size_of::<AccountMeta>()),     PARTIALLY
                (hash_ptr as *const u8, mem::size_of::<Hash>()),                    FOUND
                (data_ptr, data_len),                                               FOUND
 


accounts/ files:


00000000   5B AD 9A D9  05 00 00 00
                                     A5 00 00 00  00 00 00 00  [...............  -> DATA_LEN (165) 
00000010   FA 5A 35 34  39 06 38 15  1B 33 6A 74  2F 64 8E 48  .Z549.8..3jt/d.H
00000020   38 5D 01 54  CF 90 F1 CA  55 DA 8D E9  75 B1 5C 28  8].T....U...u.\(  -> pubkey de l'account
                                                                                    HrGhscadmYEhMu6tLqjFhTz3RkJm1kgzNzggh6NTopiK


00000030   F0 1D 1F 00  00 00 00 00
                                     89 00 00 00  00 00 00 00  ................  -> epoch (137)

00000040   06 DD F6 E1  D7 65 A1 93  D9 CB E1 46  CE EB 79 AC  .....e.....F..y.
00000050   1C B4 85 ED  5F 5B 37 91  3A 8C F5 85  7E FF 00 A9  ...._[7.:...~...  -> owner key
                                                                                    TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA

00000060   00 2F 0B 00  00 00 00 00

                                     DD E2 E1 63  E0 83 35 77  ./.........c..5w
00000070   BB B4 C9 5F  63 45 30 B5  82 28 07 FF  B4 0B 8E CF  ..._cE0..(......
00000080   65 02 B9 23  1E E7 F7 0E

data gathered on chain:
0b3338a0ab2cc841d5b014bc6a3cf756
291874b319c9517d9bbfa9e4e9661ef9
dd909cdfa3ba7ee7ae409bcb8c5f4a0c
7f157d8d6d982476c0088ac0997f2651
40420f00000000000000000000000000
00000000000000000000000000000000
00000000000000000000000001000000
00000000000000000000000000000000
000100000008a13fb5c9e7bc18aef6d4
ec2e5bca9fb0b8c329c32bdf2baae912
5aa3191cd3


                                     0B 33 38 A0  AB 2C C8 41  e..#.....38..,.A
00000090   D5 B0 14 BC  6A 3C F7 56  29 18 74 B3  19 C9 51 7D  ....j<.V).t...Q}
000000A0   9B BF A9 E4  E9 66 1E F9  DD 90 9C DF  A3 BA 7E E7  .....f........~.
000000B0   AE 40 9B CB  8C 5F 4A 0C  7F 15 7D 8D  6D 98 24 76  .@..._J...}.m.$v
000000C0   C0 08 8A C0  99 7F 26 51  40 42 0F 00  00 00 00 00  ......&Q@B......
000000D0   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00  ................
000000E0   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00  ................
000000F0   00 00 00 00  01 00 00 00  00 00 00 00  00 00 00 00  ................
00000100   00 00 00 00  00 00 00 00  00 01 00 00  00 08 A1 3F  ...............?
00000110   B5 C9 E7 BC  18 AE F6 D4  EC 2E 5B CA  9F B0 B8 C3  ..........[.....
00000120   29 C3 2B DF  2B AA E9 12  5A A3 19 1C  D3 00 00 00  ).+.+...Z.......  -> ends the data we found + 3 bytes of padding



-------------------


00000130   2C AB 9A D9  05 00 00 00  A5 00 00 00  00 00 00 00  ,...............

00000140   5D 56 04 A6  82 9A F5 AF  39 6B 58 65  A0 BC F9 3F  ]V......9kXe...?
00000150   B5 21 F6 F8  66 99 C8 F2  F3 21 AD F8  98 12 D6 9E  .!..f....!...... -> 7HLzU2jFMGjHQm53kHJGc1WmngjfkjmptqoMYLuXnrh3 ?

00000160   F0 1D 1F 00  00 00 00 00  89 00 00 00  00 00 00 00  ................
00000170   06 DD F6 E1  D7 65 A1 93  D9 CB E1 46  CE EB 79 AC  .....e.....F..y.
00000180   1C B4 85 ED  5F 5B 37 91  3A 8C F5 85  7E FF 00 A9  ...._[7.:...~...
00000190   00 2F 0B 00  00 00 00 00  AC D9 72 97  84 41 F8 80  ./........r..A..
000001A0   0B 2F 4D EC  35 A3 21 39  02 8E FB 49  4E B8 A6 70  ./M.5.!9...IN..p
000001B0   F1 A1 64 48  60 F6 CA C9  0B 33 38 A0  AB 2C C8 41  ..dH`....38..,.A
000001C0   D5 B0 14 BC  6A 3C F7 56  29 18 74 B3  19 C9 51 7D  ....j<.V).t...Q}     -> end of data corresponding to what he has



000001D0   9B BF A9 E4  E9 66 1E F9  5D 56 04 A6  82 9A F5 AF  .....f..]V......
000001E0   39 6B 58 65  A0 BC F9 3F  B5 21 F6 F8  66 99 C8 F2  9kXe...?.!..f...
000001F0   F3 21 AD F8  98 12 D6 9E  00 00 00 00  00 00 00 00  .!..............
00000200   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00  ................
00000210   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00  ................
00000220   00 00 00 00  01 00 00 00  00 00 00 00  00 00 00 00  ................
00000230   00 00 00 00  00 00 00 00  00 01 00 00  00 08 A1 3F  ...............?
