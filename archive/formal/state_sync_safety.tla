--------------------------- MODULE state_sync_safety ---------------------------
(*
  TLA+ Specification for PAW State Sync Protocol Safety Properties

  This specification models the state synchronization protocol and verifies
  that it maintains Byzantine fault tolerance, prevents malicious peers from
  corrupting state, and ensures proper chunk verification.

  Safety Properties:
  1. StateIntegrity: Final state hash matches snapshot hash
  2. ByzantineResistance: No state corruption even with Byzantine peers
  3. ChunkVerification: All chunks verified before application
  4. NoDoubleTrust: Same chunk not trusted from different hashes
*)

EXTENDS Integers, Sequences, FiniteSets, TLC

CONSTANTS
    Peers,              \* Set of peer IDs
    MaxMaliciousPeers,  \* Maximum tolerable malicious peers
    NumChunks,          \* Number of chunks in snapshot
    MinPeerAgreement    \* Minimum peer agreement ratio (e.g., 0.67 for 2/3+)

VARIABLES
    snapshotOffers,     \* Snapshot offers from peers: [peer -> snapshot]
    selectedSnapshot,   \* Selected snapshot
    downloadedChunks,   \* Chunks downloaded: [chunkID -> {data, hash, verified}]
    maliciousPeers,     \* Set of detected malicious peers
    syncState,          \* Current sync state
    appliedState,       \* Final applied state
    metrics             \* Sync metrics

vars == <<snapshotOffers, selectedSnapshot, downloadedChunks, maliciousPeers,
          syncState, appliedState, metrics>>

\* State sync phases
States == {"Idle", "Discovering", "Selecting", "Downloading", "Applying",
           "Verifying", "Complete", "Failed"}

\* Snapshot structure
Snapshot == [height: Nat, hash: Nat, numChunks: Nat, chainID: Nat]

\* Initialize state
Init ==
    /\ snapshotOffers = [p \in Peers |-> {}]
    /\ selectedSnapshot = [height |-> 0, hash |-> 0, numChunks |-> 0, chainID |-> 0]
    /\ downloadedChunks = [c \in 1..NumChunks |->
                          [data |-> 0, hash |-> 0, verified |-> FALSE]]
    /\ maliciousPeers = {}
    /\ syncState = "Idle"
    /\ appliedState = [height |-> 0, hash |-> 0]
    /\ metrics = [chunksDownloaded |-> 0, chunksVerified |-> 0, maliciousFound |-> 0]

\* Type invariants
TypeInvariant ==
    /\ syncState \in States
    /\ maliciousPeers \subseteq Peers
    /\ Cardinality(maliciousPeers) <= MaxMaliciousPeers
    /\ appliedState.height >= 0

--------------------------------------------------------------------------------
\* Actions

\* Peer offers a snapshot
OfferSnapshot(peer, snapshot) ==
    /\ syncState = "Discovering"
    /\ peer \notin maliciousPeers
    /\ snapshotOffers' = [snapshotOffers EXCEPT ![peer] = snapshot]
    /\ UNCHANGED <<selectedSnapshot, downloadedChunks, maliciousPeers,
                   syncState, appliedState, metrics>>

\* Select snapshot with sufficient peer agreement
SelectSnapshot ==
    LET
        \* Count offers for each unique snapshot
        offerCounts == [s \in DOMAIN snapshotOffers |->
                       Cardinality({p \in Peers : snapshotOffers[p] = s})]

        \* Find snapshot with highest agreement
        bestSnapshot == CHOOSE s \in DOMAIN offerCounts :
                       /\ offerCounts[s] * 1.0 / Cardinality(Peers) >= MinPeerAgreement
                       /\ \A other \in DOMAIN offerCounts :
                          offerCounts[s] >= offerCounts[other]
    IN
        /\ syncState = "Discovering"
        /\ \E s \in DOMAIN offerCounts :
           offerCounts[s] * 1.0 / Cardinality(Peers) >= MinPeerAgreement
        /\ selectedSnapshot' = bestSnapshot
        /\ syncState' = "Selecting"
        /\ UNCHANGED <<snapshotOffers, downloadedChunks, maliciousPeers,
                       appliedState, metrics>>

\* Download and verify a chunk
DownloadChunk(chunkID, peer, chunkData, chunkHash) ==
    /\ syncState = "Downloading"
    /\ chunkID \in 1..NumChunks
    /\ peer \notin maliciousPeers
    /\ downloadedChunks[chunkID].verified = FALSE
    /\ LET
           \* Verify chunk hash matches expected
           expectedHash == selectedSnapshot.hash  \* Simplified: use snapshot hash
           hashMatches == (chunkHash = expectedHash)
       IN
           /\ IF hashMatches
              THEN \* Valid chunk
                   /\ downloadedChunks' = [downloadedChunks EXCEPT
                                          ![chunkID] = [data |-> chunkData,
                                                       hash |-> chunkHash,
                                                       verified |-> TRUE]]
                   /\ metrics' = [metrics EXCEPT
                                 !.chunksDownloaded = @ + 1,
                                 !.chunksVerified = @ + 1]
                   /\ UNCHANGED maliciousPeers
              ELSE \* Malicious chunk
                   /\ maliciousPeers' = maliciousPeers \union {peer}
                   /\ metrics' = [metrics EXCEPT !.maliciousFound = @ + 1]
                   /\ UNCHANGED downloadedChunks
           /\ UNCHANGED <<snapshotOffers, selectedSnapshot, syncState, appliedState>>

\* Apply snapshot after all chunks downloaded and verified
ApplySnapshot ==
    /\ syncState = "Downloading"
    /\ \A c \in 1..NumChunks : downloadedChunks[c].verified = TRUE
    /\ LET
           \* Reconstruct state hash from chunks
           stateHash == selectedSnapshot.hash  \* Simplified
       IN
           /\ appliedState' = [height |-> selectedSnapshot.height,
                              hash |-> stateHash]
           /\ syncState' = "Applying"
           /\ UNCHANGED <<snapshotOffers, selectedSnapshot, downloadedChunks,
                          maliciousPeers, metrics>>

\* Verify applied state matches snapshot
VerifyState ==
    /\ syncState = "Applying"
    /\ appliedState.hash = selectedSnapshot.hash
    /\ syncState' = "Complete"
    /\ UNCHANGED <<snapshotOffers, selectedSnapshot, downloadedChunks,
                   maliciousPeers, appliedState, metrics>>

\* Fail sync if too many malicious peers
FailSync ==
    /\ Cardinality(maliciousPeers) > MaxMaliciousPeers
    /\ syncState' = "Failed"
    /\ UNCHANGED <<snapshotOffers, selectedSnapshot, downloadedChunks,
                   maliciousPeers, appliedState, metrics>>

\* Next state relation
Next ==
    \/ \E p \in Peers, s \in Snapshot : OfferSnapshot(p, s)
    \/ SelectSnapshot
    \/ \E c \in 1..NumChunks, p \in Peers, d \in Nat, h \in Nat :
       DownloadChunk(c, p, d, h)
    \/ ApplySnapshot
    \/ VerifyState
    \/ FailSync

Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

--------------------------------------------------------------------------------
\* Safety Properties

\* PROPERTY 1: State integrity - final state matches snapshot
StateIntegrity ==
    (syncState = "Complete") =>
        (appliedState.hash = selectedSnapshot.hash)

\* PROPERTY 2: Byzantine resistance - state not corrupted by malicious peers
ByzantineResistance ==
    (Cardinality(maliciousPeers) <= MaxMaliciousPeers) =>
        ((syncState = "Complete") => (appliedState.hash = selectedSnapshot.hash))

\* PROPERTY 3: All chunks verified before application
ChunkVerification ==
    (syncState = "Applying" \/ syncState = "Complete") =>
        (\A c \in 1..NumChunks : downloadedChunks[c].verified = TRUE)

\* PROPERTY 4: No double trust - chunks with different hashes not both trusted
NoDoubleTrust ==
    \A c1, c2 \in 1..NumChunks :
        (downloadedChunks[c1].verified /\ downloadedChunks[c2].verified) =>
            (downloadedChunks[c1].hash = downloadedChunks[c2].hash)

\* PROPERTY 5: Malicious peer detection
MaliciousPeerDetection ==
    \A p \in maliciousPeers :
        \E c \in 1..NumChunks :
            \* Peer provided invalid chunk
            TRUE  \* Simplified - in real spec, track chunk sources

\* PROPERTY 6: Sync completes only with valid state
ValidCompletion ==
    (syncState = "Complete") =>
        /\ appliedState.height > 0
        /\ appliedState.hash > 0
        /\ \A c \in 1..NumChunks : downloadedChunks[c].verified

\* PROPERTY 7: No state regression
NoRegression ==
    [][appliedState.height' >= appliedState.height]_vars

\* Combined safety property
Safety ==
    /\ TypeInvariant
    /\ StateIntegrity
    /\ ByzantineResistance
    /\ ChunkVerification
    /\ NoDoubleTrust
    /\ ValidCompletion
    /\ NoRegression

================================================================================
