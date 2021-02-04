# dfuse-solana


- We need to keep a local history of all fill events for a list of given markets... 
    1) load all the markets into cache getAccountOnEach
    2) we need to watch all the event address of each market.... (1 market -> 1 event queue address)
    3) we need to filter out the fills of all the incoming events and store that per market
    4) we need to persist this cache wise
    
- When a new  graphql subscription occurs we need to retrieve historically all hist transactions
- filter out the NewOrder
- Attempt to cross match a new Order & fill events per market to see if it is a recent trade... (i.e. to see if the order got filled)


Questions)

- Is there pagination on `getAccountInfo`?
    There is pagination based on offset
    
    
- 1) load a cache/store from disk (sts)    
- 2) read jsonl with all markets, and add the ones that aren't the cache
- 3)




type Store struct {
    map[string] MarketMeta {
           
    }
}


store Market version => data decoder

getAcountInfo(eventAddress)
-> HEAD EVENT, EVENT, EVENT, EVENT, EVENT, .....EVENT


->load markets
-> go routine
        getAccountInfo(eventAddress)
        load all events store
        ws.subscribeAccount(eventAccount)
            -> update events list
            -> update events list
            
3 min
->       
            



 


// We are not tracking the the IRR block correctly (current hard wired to -10)
//