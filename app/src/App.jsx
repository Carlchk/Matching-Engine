import logo from './logo.svg';
import { createSignal, For, createEffect, onMount, onCleanup } from "solid-js";
// import createWebsocket from "@solid-primitives/websocket";
import axios from 'axios';
import useWebsocket from './useWebSocket.jsx';



function App() {

  const connectionMap = {
    "btcusdt": 4001,
    "ethusdt": 4002
  }
  const [selectedPairs, setSelectedPairs] = createSignal("btcusdt");

  const [input, setInput] = createSignal({
    price_type: "limit",
    price: 0,
    quantity: 0,
  });


  const [asksOrders, setAsksOrders] = createSignal([]);
  const [bidsOrders, setBidsOrders] = createSignal([]);
  const [latestPrice, setLatestPrice] = createSignal(0);
  const [openOrders, setOpenOrders] = createSignal([]);
  const [historicalOrder, setHistoricalOrder] = createSignal([]);

  const [connect, disconnect, changeUrl, state, socket] = useWebsocket(
    `ws://localhost:${connectionMap[selectedPairs()]}/ws`,
    (msg) => wsHandler(msg),
    (msg) => console.log(msg.error),
    [],
    5,
    5000
  )

  function wsHandler(evt) {
    try {
      var messages = evt.data.split('\n');

      for (var i = 0; i < messages.length; i++) {
        var data = JSON.parse(messages[i]);
        switch (data.tag) {
          case "depth":
            const info = data.data;
            setAsksOrders(info.ask.reverse());
            setBidsOrders(info.bid);
            break;
          case "trade":
            setHistoricalOrder([data.data, ...historicalOrder()]);
            // remove order from open orders
            removeObjectWithId(data.data);
            break;
          case "new_order":
            // console.log("Updated")
            // setOpenOrders([data.data, ...openOrders()]);
            break;
          case "latest_price":
            setLatestPrice(data.data.latest_price);
            break;
          default:
            console.log("Unknown tag: " + data.tag);
        }
      }
    } catch (err) {
      console.log(err);
    }
  }

  function removeObjectWithId({ AskOrderId, BidOrderId }) {
    setOpenOrders(openOrders().filter(order => order.order_id !== AskOrderId));
    setOpenOrders(openOrders().filter(order => order.order_id !== BidOrderId));
  }


  function formatTime(t) {
    var d = new Date(t);
    return d.getDate() + '/' + (d.getMonth() + 1) + '/' + d.getFullYear() + ' ' + d.getHours() + ':' + d.getMinutes() + ':' + (d.getSeconds() > 10 ? d.getSeconds() : '0' + d.getSeconds());
  }

  async function submitNewOrder(type) {
    var data = input();
    data.order_type = type;
    data.pairs = selectedPairs();

    await axios.post("http://localhost:3001/new_order", data)
      .then((res) => {
        setOpenOrders([res.data, ...openOrders()]);
        console.log("order placed successfully");
      })
  }

  async function cancelOrder(orderId) {
    await axios.post(`http://localhost:${connectionMap[selectedPairs()]}/api/cancel_order`, { order_id: orderId })
      .then((res) => {
        // remove order from open orders
        if (res.status === 200) {
          setOpenOrders(openOrders().filter(order => order.order_id !== orderId));
          console.log("order cancelled successfully");
        }
      })
  }

  // get previous trading
  async function getHistoricalOrder() {
    await axios.get(`http://localhost:${connectionMap[selectedPairs()]}/api/trade_log`)
      .then((res) => {
        // remove order from open orders
        if (res.status === 200) {
          console.log(res.data)
          setHistoricalOrder(res.data.data.trade_log);
        }
      })
  }

  function handleInputChange(key, data) {
    setInput({
      ...input(),
      [key]: data,
    });
  }

  async function refreshUI() {
    setAsksOrders([]);
    setBidsOrders([]);
    setLatestPrice(0);
    setHistoricalOrder([]);
    setOpenOrders([]);
    await getHistoricalOrder();
  }

  const handleDropdownChange = (selectedValue) => {
    setSelectedPairs(selectedValue);
    changeUrl(`ws://localhost:${connectionMap[selectedValue]}/ws`);
    refreshUI();
  };

  onMount(async () => {
    connect();
    await getHistoricalOrder();
  })

  return (
    <div class="h-[calc(100vh-48px)] text-white bg-myblack">
      <div class="h-12">
        <div class="flex items-center justify-between h-full px-4">
          <h1 class="text-white text-2xl">OSL Coding Test Q2 - Matching Engine</h1>
          <div class="relative font-semibold label-color">
            <select name="price_type"
              onChange={(event) => { handleDropdownChange(event.target.value) }}
              class="block appearance-none w-full bg-white border border-gray-400 hover:border-gray-500 px-4 py-1 pr-8 rounded shadow leading-tight focus:outline-none focus:shadow-outline">
              <option value="btcusdt">BTC</option>
              <option value="ethusdt">ETH</option>
            </select>
            <div
              class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-gray-700">
              <svg class="fill-current h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20">
                <path d="M9.293 12.95l.707.707L15.657 8l-1.414-1.414L10 10.828 5.757 6.586 4.343 8z" />
              </svg>
            </div>
          </div>
        </div>
      </div>
      <div class="parent p-2 h-full">
        <div class="div1 orderbook-layout p-2">
          <div class="h-full font-medium">
            <div class="text-white font-semibold">
              Order Book
            </div>
            <div class="flex flex-row justify-between text-sm font-semibold label-color">
              <div class="min-w-[33%] text-left">Price(USDT)</div>
              <div class="min-w-[33%] text-right">Amount</div>
              <div class="min-w-[33%] text-right pr-2">Total</div>
            </div>
            <div id="asks-order-book" class="buy-color text-sm">
              <For each={asksOrders()}>{(askOrder, i) =>
                <div class="flex flex-row justify-between">
                  <div class="min-w-[33%] text-left">${askOrder[0]}</div>
                  <div class="min-w-[33%] text-right font-normal">{askOrder[1]}</div>
                  <div class="min-w-[33%] text-right font-normal pr-2">${askOrder[0] * askOrder[1]}</div>
                </div>
              }</For>
            </div>
            <div id="latest-price" class="py-2 font-semibold">{latestPrice}</div>
            <div id="bids-order-book" class="sell-color text-sm">
              <For each={bidsOrders()}>{(bidOrder, i) =>
                <div class="flex flex-row justify-between">
                  <div class="min-w-[33%] text-left">${bidOrder[0]}</div>
                  <div class="min-w-[33%] text-right font-normal">{bidOrder[1]}</div>
                  <div class="min-w-[33%] text-right font-normal pr-2">${bidOrder[0] * bidOrder[1]}</div>
                </div>
              }</For>
            </div>
          </div>
        </div>
        <div class="div2 p-2">
          <div class="text-white font-semibold">
            Open Orders
          </div>
          <div class="flex flex-row justify-between text-sm font-semibold label-color">
            <div class="min-w-[16%] text-left">Type</div>
            <div class="min-w-[16%] text-right">Price</div>
            <div class="min-w-[16%] text-right">Quantity/Filled</div>
            <div class="min-w-[16%] text-right pr-2">Total</div>
            <div class="min-w-[16%] text-right pr-2">Timestamp</div>
            <div class="min-w-[16%] text-right pr-2">Actions</div>
          </div>
          <div id="open-orders-panel" class="text-sm">
            <For each={openOrders()}>{(order, i) =>
              <div orderId={order.order_id} class="flex flex-row justify-between">
                <div class="min-w-[16%] text-left">{order.price_type}</div>
                <div class="min-w-[16%] text-right font-normal">${order.price}</div>
                <div class="min-w-[16%] text-right font-normal pr-2">{order.quantity}/0</div>
                <div class="min-w-[16%] text-right font-normal pr-2">{order.amountStr}</div>
                <div class="min-w-[16%] text-right font-normal pr-2">{formatTime(order.create_time / 1e6)}</div>
                <div class="min-w-[16%] text-right font-normal pr-2">
                  <a class="cancel" href="javascript:;" onClick={() => { cancelOrder(order.order_id) }}>Cancel</a>
                </div>
              </div>
            }</For>
          </div>
        </div>
        <div class="div3 p-2">
          <div class="text-white font-semibold">
            Order History
          </div>
          <div class="flex flex-row justify-between text-sm font-semibold label-color">
            <div class="min-w-[25%] text-left">Price</div>
            <div class="min-w-[25%] text-right">Quantity</div>
            <div class="min-w-[25%] text-right pr-2">Total</div>
            <div class="min-w-[25%] text-right pr-2">Timestamp</div>
          </div>
          <div id="history-orders-panel" class="text-sm">
            <For each={historicalOrder()}>
              {(order, i) =>
                <div class="flex flex-row justify-between">
                  <div class="min-w-[25%] text-left">${order.TradePrice}</div>
                  <div class="min-w-[25%] text-right font-normal">{order.TradeQuantity}</div>
                  <div class="min-w-[25%] text-right font-normal pr-2">{order.TradeAmount}</div>
                  <div class="min-w-[25%] text-right font-normal pr-2">{formatTime(order.TradeTime / 1e6)}</div>
                </div>
              }
            </For>
          </div>
        </div>
        <div class="div4">
          <div class="flex flex-col justify-around text-sm font-semibold label-color">
            <div class="inline-block relative w-64 m-2">
              <label class="block uppercase tracking-wide label-color text-xs font-bold mb-2"
                for="grid-last-name">
                Order Type
              </label>
              <div class="relative">
                <select name="price_type"
                  onChange={(event) => { handleInputChange("price_type", event.target.value) }}
                  class="block appearance-none w-full bg-white border border-gray-400 hover:border-gray-500 px-4 py-2 pr-8 rounded shadow leading-tight focus:outline-none focus:shadow-outline">
                  <option value="limit">Limted Order</option>
                  <option value="market" disabled>Market Order</option>
                </select>
                <div
                  class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-gray-700">
                  <svg class="fill-current h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20">
                    <path d="M9.293 12.95l.707.707L15.657 8l-1.414-1.414L10 10.828 5.757 6.586 4.343 8z" />
                  </svg>
                </div>
              </div>
            </div>
            <div class="inline-block relative w-64 m-2">
              <label class="block uppercase tracking-wide label-color text-xs font-bold mb-2"
                for="grid-last-name">
                Price
              </label>
              <input type="text" name="price" required
                onChange={(event) => { handleInputChange("price", event.target.value) }}
                class="w-full px-3 py-2 text-gray-700 border rounded focus:outline-none" placeholder="0.00" />
            </div>
            <div class="inline-block relative w-64 m-2">
              <label class="block uppercase tracking-wide label-color text-xs font-bold mb-2"
                for="grid-last-name">
                Qty
              </label>
              <input type="text" name="quantity" required
                onChange={(event) => { handleInputChange("quantity", event.target.value) }}
                class="w-full px-3 py-2 text-gray-700 border rounded focus:outline-none" placeholder="0.00" />
            </div>
            <div class="flex flex-row justify-start text-sm font-semibold label-color">
              <button
                onClick={() => { submitNewOrder("ask") }}
                class="bg-buy-color hover:bg-buy-color text-white font-bold py-2 px-4 m-2 rounded opt sell">
                BUY
              </button>
              <button
                onClick={() => { submitNewOrder("bid") }}
                class="bg-sell-color hover:bg-buy-color text-white font-bold py-2 px-4 m-2 rounded opt buy">
                SELL
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
