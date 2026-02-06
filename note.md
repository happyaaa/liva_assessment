Requirements
* Fraud detection must be O(log n) per user (n = recordings per user)
* Concurrent requests must not corrupt data
* Write tests demonstrating correct fraud handling

Go

API
POST /recording/end   → credit earnings, detect fraud
fraud locks binary search 

GET  /balance/:userId → return balance
locks

POST /withdraw        → withdraw funds

Stack: TypeScript, Go, or Python
In-memory storage. You decide the request/response form



In-memory map go mutex

User: log(n) binary search build in , locks

Concurrent: locks, sorting (userid, userid+starttime)

$1/hour (pro-rated to the minute, round down to nearest minute):
100 cents (int) floor()

