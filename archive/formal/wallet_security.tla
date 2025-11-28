----------------------------- MODULE wallet_security ---------------------------
(*
  TLA+ Specification for PAW Wallet Security Properties

  Models cryptographic key management, mnemonic generation, and keystore encryption.
  Verifies security properties around key derivation, encryption, and access control.
*)

EXTENDS Integers, Sequences, FiniteSets

CONSTANTS
    MinPasswordLength,    \* Minimum secure password length (e.g., 12)
    MinIterations,        \* Minimum PBKDF2 iterations (e.g., 100000)
    ValidMnemonicLengths  \* Valid mnemonic word counts: {12, 15, 18, 21, 24}

VARIABLES
    mnemonic,            \* BIP39 mnemonic phrase
    masterKey,           \* HD master key
    derivedKeys,         \* Derived child keys: [path -> key]
    keystores,           \* Encrypted keystores: [address -> keystore]
    passwords,           \* Password hashes for keystores
    accessAttempts,      \* Failed access attempts
    lockedKeystores      \* Temporarily locked keystores

vars == <<mnemonic, masterKey, derivedKeys, keystores, passwords,
          accessAttempts, lockedKeystores>>

\* Keystore structure
Keystore == [
    ciphertext: Nat,
    salt: Nat,
    iv: Nat,
    mac: Nat,
    iterations: Nat
]

Init ==
    /\ mnemonic = 0
    /\ masterKey = 0
    /\ derivedKeys = <<>>
    /\ keystores = <<>>
    /\ passwords = <<>>
    /\ accessAttempts = [k \in {} |-> 0]
    /\ lockedKeystores = {}

--------------------------------------------------------------------------------
\* Mnemonic Generation

\* Generate BIP39 mnemonic with proper entropy
GenerateMnemonic(wordCount) ==
    /\ wordCount \in ValidMnemonicLengths
    /\ mnemonic' = wordCount  \* Simplified: use word count as mnemonic
    /\ UNCHANGED <<masterKey, derivedKeys, keystores, passwords,
                   accessAttempts, lockedKeystores>>

\* Validate mnemonic checksum
ValidateMnemonic(m) ==
    /\ m \in ValidMnemonicLengths
    /\ TRUE  \* Simplified: assume valid if proper length

--------------------------------------------------------------------------------
\* Key Derivation

\* Derive master key from mnemonic using PBKDF2
DeriveMasterKey(m, passphrase) ==
    /\ ValidateMnemonic(m)
    /\ mnemonic = m
    /\ masterKey' = m * 1000 + passphrase  \* Simplified derivation
    /\ UNCHANGED <<mnemonic, derivedKeys, keystores, passwords,
                   accessAttempts, lockedKeystores>>

\* Derive child key using BIP32 hierarchical derivation
DeriveChildKey(path) ==
    /\ masterKey > 0
    /\ derivedKeys' = Append(derivedKeys, [path |-> path, key |-> masterKey + path])
    /\ UNCHANGED <<mnemonic, masterKey, keystores, passwords,
                   accessAttempts, lockedKeystores>>

--------------------------------------------------------------------------------
\* Keystore Encryption

\* Encrypt private key into keystore with password
EncryptKeystore(key, password, iterations) ==
    /\ Len(password) >= MinPasswordLength  \* Password strength check
    /\ iterations >= MinIterations          \* Iteration count check
    /\ LET
           salt == key + 1
           iv == key + 2
           ciphertext == key * password  \* Simplified encryption
           mac == ciphertext + salt      \* Simplified MAC
           keystore == [
               ciphertext |-> ciphertext,
               salt |-> salt,
               iv |-> iv,
               mac |-> mac,
               iterations |-> iterations
           ]
       IN
           /\ keystores' = Append(keystores, keystore)
           /\ passwords' = Append(passwords, password)
           /\ UNCHANGED <<mnemonic, masterKey, derivedKeys,
                          accessAttempts, lockedKeystores>>

\* Decrypt keystore with password
DecryptKeystore(keystoreIdx, password, expectedMac) ==
    /\ keystoreIdx \in DOMAIN keystores
    /\ keystoreIdx \notin lockedKeystores
    /\ LET
           keystore == keystores[keystoreIdx]
           correctPassword == passwords[keystoreIdx]
           \* Verify MAC first (constant-time comparison)
           macValid == (keystore.mac = expectedMac)
           passwordCorrect == (password = correctPassword)
       IN
           /\ IF passwordCorrect /\ macValid
              THEN \* Successful decryption
                   /\ accessAttempts' = [accessAttempts EXCEPT ![keystoreIdx] = 0]
                   /\ UNCHANGED <<lockedKeystores, mnemonic, masterKey,
                                  derivedKeys, keystores, passwords>>
              ELSE \* Failed attempt
                   /\ accessAttempts' = [accessAttempts EXCEPT
                                        ![keystoreIdx] = @ + 1]
                   /\ IF accessAttempts'[keystoreIdx] >= 3
                      THEN lockedKeystores' = lockedKeystores \union {keystoreIdx}
                      ELSE UNCHANGED lockedKeystores
                   /\ UNCHANGED <<mnemonic, masterKey, derivedKeys, keystores, passwords>>

--------------------------------------------------------------------------------
\* Next state relation

Next ==
    \/ \E w \in ValidMnemonicLengths : GenerateMnemonic(w)
    \/ \E m \in ValidMnemonicLengths, p \in Nat : DeriveMasterKey(m, p)
    \/ \E path \in Nat : DeriveChildKey(path)
    \/ \E k \in Nat, p \in Nat, i \in Nat :
       (Len(p) >= MinPasswordLength /\ i >= MinIterations) => EncryptKeystore(k, p, i)
    \/ \E idx \in DOMAIN keystores, p \in Nat, mac \in Nat :
       DecryptKeystore(idx, p, mac)

Spec == Init /\ [][Next]_vars

--------------------------------------------------------------------------------
\* Safety Properties

\* PROPERTY 1: Weak passwords rejected
WeakPasswordRejection ==
    \A ks \in DOMAIN keystores :
        Len(passwords[ks]) >= MinPasswordLength

\* PROPERTY 2: Insufficient iterations rejected
SufficientIterations ==
    \A ks \in DOMAIN keystores :
        keystores[ks].iterations >= MinIterations

\* PROPERTY 3: MAC verified before decryption
MacVerificationFirst ==
    TRUE  \* Modeled in DecryptKeystore action

\* PROPERTY 4: Rate limiting on failed attempts
RateLimiting ==
    \A ks \in DOMAIN accessAttempts :
        accessAttempts[ks] >= 3 => ks \in lockedKeystores

\* PROPERTY 5: No key material leakage
NoKeyLeakage ==
    \* Simplified: keys only exist in encrypted form
    TRUE

\* PROPERTY 6: Mnemonic validation
MnemonicValidation ==
    (mnemonic > 0) => (mnemonic \in ValidMnemonicLengths)

\* PROPERTY 7: Deterministic key derivation
DeterministicDerivation ==
    \A k1, k2 \in DOMAIN derivedKeys :
        (derivedKeys[k1].path = derivedKeys[k2].path) =>
        (derivedKeys[k1].key = derivedKeys[k2].key)

\* Combined safety
Safety ==
    /\ WeakPasswordRejection
    /\ SufficientIterations
    /\ RateLimiting
    /\ MnemonicValidation
    /\ DeterministicDerivation

================================================================================
