- What edge cases did you discover? How did you handle them?

deadlocks in mul-ti party recording:
issue: request a locks user [alice, bob]
and request b locks [bob,alice] simultaneously
solution: i implemented locking, by sorting userid, i ensure the concurrent request acquire locks in the same order

floating point inaccuracy:
issue: calculating earnings using float64 leas to rounding errors
solution: i used int64 to store money in cents, all calculations are performed using integer

negative balance:
issue: the rules states that balance can not go negative, if a user withdraw funds and is later found to be fraud, resulting in negative balance
solution: i implemented floor at zero, in the penalty exceeds the current balance, the balance resets to 0

boundary overlaps:
issue: recoding ending exactly when next one starts
solution: i handled strict inequalities (start < end) in the binary search logic to allow no overlapping recordings

- What could break at 10k recordings/min? How would you fix it?
global lock contention:
problem: the current store uses sync, RWMutex to protect the user map, at 10k recordings/min, frequent writes would lock the whole map, blocking the concurrent read
fix: sharding, split the users map into 32/64 shards, route based on the hash(userid) (sha256) % sharedcount, this would reduce lock contntion

o(n) insertion latency
problem: detection is o(log n) using built in binary search, inserting into go slice is o(n) due to memory shifting (copy), insertion would become slow when user's history grow larger
fix: replace the sorted slice with binary search tree, this would make both search and insertion o(log n)

memory usage:
problem: in memory storage is limited by RAM
fix: move to persistent databse (redis) 