use tonic::{transport::Server, Request, Response, Status};
use rand::prelude::*;

use log::info;
use tokio::sync::mpsc;
use tokio::sync::Mutex;
use std::sync::Arc;

use stream::streamer_server::{Streamer, StreamerServer};
use stream::{ListenReq, ListenEvent};

const MAX_ITEMS: usize = 10000;

pub mod stream {
    tonic::include_proto!("stream");
}

#[derive(Debug, Clone)]
struct Item {
    id: u64,
    state: bool,
}

impl Item {
    pub fn as_pb_event(&self) -> ListenEvent {
        ListenEvent {
            id: self.id,
            new_state: if self.state { 1 } else { -1 },
        }
    }
}

#[derive(Debug, Clone)]
struct ItemEvent {
    id: u64,
    new_state: bool,
}

impl ItemEvent {
    pub fn as_pb_event(&self) -> ListenEvent {
        ListenEvent {
            id: self.id,
            new_state: if self.new_state { 1 } else { -1 },
        }
    }
}

#[derive(Debug)]
struct Listener {
    queue: tokio::sync::mpsc::Sender<ItemEvent>,
}

#[derive(Debug, Default)]
struct Stream {
    listeners: Vec<Listener>,
    items: Vec<Item>,
}

#[derive(Debug, Default)]
pub struct StreamX {
    inner: Arc<Mutex<Stream>>,
}

#[tonic::async_trait]
impl Streamer for StreamX {
    type ListenStream = mpsc::Receiver<Result<ListenEvent, Status>>;

    async fn listen(
        &self,
        _request: Request<ListenReq>,
    ) -> Result<Response<Self::ListenStream>, Status> {
        //println!("streaming!");
        let (mut tx, rx) = mpsc::channel(4);
        let ss = self.inner.clone();

        tokio::spawn(async move {
            // examine some internal data in StreamX, calling `tx.send().await`
            let mut lrx = {
                let mut ss = ss.lock().await;
                for i in ss.items.iter() {
                    tx.send(Ok(i.as_pb_event())).await.unwrap();
                }

                let (ltx, lrx) = mpsc::channel(4);

                ss.listeners.push(Listener {
                    queue: ltx,
                });

                lrx
            };

            loop {
                let v = lrx.recv().await.unwrap();   
                let n = v.as_pb_event();

                match tx.send(Ok(n)).await {
                    Ok(_) => {},
                    Err(e) => {
                        //eprintln!("closing listener: {}", e);
                        // implicitly closes our end of the channel
                        return;
                    },
                }
            }
        });

        Ok(Response::new(rx))
    }
}

async fn send_event(ss: &mut Stream,event: ItemEvent) {

    for ix in 0..ss.listeners.len() {
        match ss.listeners[ix].queue.send(event.clone()).await {
            Ok(_) => {},
            Err(_) => {
                // other end closed down, remove this listener
                //println!("closing listener");
                ss.listeners.remove(ix);
            },
        }
    }
}


async fn generate(sa: Arc<Mutex<Stream>>) {
    let mut next_id = 1;

    let mut events_total = 0;
    let mut since_last_print = 0;
    let mut start_time = tokio::time::Instant::now(); 
    let mut print_time = start_time;

    loop {
        tokio::time::delay_for(std::time::Duration::from_millis(1)).await;

        {
            let mut sz = sa.lock().await;
            let ss : &mut Stream = &mut *sz;

            if ss.items.len() < MAX_ITEMS {
                if rand::random() {
                    //println!("adding new item {}", next_id);
                    // locks?
                    let i = Item {
                        id: next_id,
                        state: true,
                    };
                    next_id += 1;

                    ss.items.push(i.clone());

                    since_last_print += 1;
                    events_total += 1;
                    send_event(ss, ItemEvent {
                        id: i.id,
                        new_state: i.state,
                    }).await;
                }
            }

            if ss.items.len() != 0 {
                let idx = rand::thread_rng().gen_range(0, ss.items.len());
                let i = &mut ss.items[idx];
                let new_state = rand::random();
                if i.state != new_state{
                    //println!("item change: {} to {}", i.id, new_state);
                    i.state = new_state;
                    let e = ItemEvent {
                        id: i.id,
                        new_state: i.state,
                    };
                    since_last_print += 1;
                    events_total += 1;
                    send_event(ss, e).await;
                }
            }

            let new_time = tokio::time::Instant::now();
            let print_duration = tokio::time::Duration::from_secs(1);
            let time_diff = new_time - print_time;
            if time_diff > print_duration {
                let total_time = new_time - start_time;
                info!("events emitted: {} ({}/s), in last {:?}: {} ({}/s)",
                    events_total, (events_total / total_time.as_secs()),
                    time_diff, since_last_print, (since_last_print / time_diff.as_secs())
                    );

                print_time = new_time;
                since_last_print = 0;
            }
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    env_logger::init();

    let addr = "[::]:7777".parse().unwrap();

    let sx = StreamX::default();

    let sz = sx.inner.clone();
    let svc = StreamerServer::new(sx);

    let serv = Server::builder().add_service(svc).serve(addr);

    let gen = tokio::spawn(async move {
        generate(sz).await
    });

    let (a, _) = tokio::join!(serv, gen);
    a.unwrap();

    Ok(())
}
