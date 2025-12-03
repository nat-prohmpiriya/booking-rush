# Known Issues & Challenges

## Pending Decisions
- เลือก Library Kafka ตัวไหนดี? (sarama vs segmentio/kafka-go) -> *Decision: ใช้ segmentio/kafka-go เพราะ API ง่ายกว่า*
- การทำ Distributed Lock จำเป็นไหม หรือแค่ Lua Script พอ? -> *Tentative: Lua Script พอสำหรับ Single Redis Node แต่ถ้า Cluster อาจต้องดู Redlock*

## Active Bugs
- (ยังไม่มี เพราะยังไม่เริ่มโค้ด)