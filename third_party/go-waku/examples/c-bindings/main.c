
#include "main.h"
#include "base64.h"
#include "libgowaku.h"
#include "nxjson.c"
#include <inttypes.h>
#include <unistd.h>

char *alicePrivKey =
    "0x4f012057e1a1458ce34189cb27daedbbe434f3df0825c1949475dec786e2c64e";
char *alicePubKey =
    "0x0440f05847c4c7166f57ae8ecaaf72d31bddcbca345e26713ca9e26c93fb8362ddcd5ae7"
    "f4533ee956428ad08a89cd18b234c2911a3b1c7fbd1c0047610d987302";

char *bobPrivKey =
    "0xb91d6b2df8fb6ef8b53b51b2b30a408c49d5e2b530502d58ac8f94e5c5de1453";
char *bobPubKey =
    "0x045eef61a98ba1cf44a2736fac91183ea2bd86e67de20fe4bff467a71249a8a0c05f795d"
    "d7f28ced7c15eaa69c89d4212cc4f526ca5e9a62e88008f506d850cccd";

void on_error(int ret, const char *result, void *user_data)
{
  if (ret == 0)
  {
    return;
  }

  printf("function execution failed. Returned code: %d, %s\n", ret, result);
  exit(1);
}

void on_response(int ret, const char *result, void *user_data)
{
  if (ret != 0)
  {
    printf("function execution failed. Returned code: %d, %s\n", ret, result);
    exit(1);
  }

  if (user_data == NULL)
    return;

  char **data_ref = (char **)user_data;
  size_t len = strlen(result);

  if (*data_ref != NULL)
    free(*data_ref);

  *data_ref = malloc(len * sizeof(char) + 1);
  strcpy(*data_ref, result);
}

void callBack(int ret, const char *signal, void *user_data)
{
  // This callback will be executed each time a new message is received

  // Example signal:
  /*{
      "nodeId":1,
      "type":"message",
      "event":{
        "pubsubTopic":"/waku/2/default-waku/proto",
        "messageID":"0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
        "wakuMessage":{
          "payload":"BPABASUqWgRkgp73aW/FHIyGtJDYnStvaQvCoX9MdaNsOH39Vet0em6ipZc3lZ7kK9uFFtbJgIWfRaqTxSRjiFOPx88gXt1JeSm2SUwGSz+1gh2xTy0am8tXkc8OWSSjamdkEbXuVgAueLxHOnV3xlGwYt7nx2G5DWYqUu1BXv4yWHPOoiH2yx3fxX0OajgKGBwiMbadRNUuAUFPRM90f+bzG2y22ssHctDV/U6sXOa9ljNgpAx703Q3WIFleSRozto7ByNAdRFwWR0RGGV4l0btJXM7JpnrYcVC24dB0tJ3HVWuD0ZcwOM1zTL0wwc0hTezLHvI+f6bHSzsFGcCWIlc03KSoMjK1XENNL4dtDmSFI1DQCGgq09c2Bc3Je3Ci6XJHu+FP1F1pTnRzevv2WP8FSBJiTXpmJXdm6evB7V1Xxj4QlzQDvmHLRpBOL6PSttxf1Dc0IwC6BfZRN5g0dNmItNlS2pcY1MtZLxD5zpj",
          "contentTopic":"ABC",
          "version":1,
          "timestamp":1647826358000000000
        }
      }
    }*/

  if (ret != 0)
  {
    printf("function execution failed. Returned code: %d\n", ret);
    exit(1);
  }

  const nx_json *json = nx_json_parse((char *)signal, 0);
  const char *type = nx_json_get(json, "type")->text_value;

  if (strcmp(type, "message") == 0)
  {
    const nx_json *wakuMsgJson =
        nx_json_get(nx_json_get(json, "event"), "wakuMessage");
    const char *contentTopic =
        nx_json_get(wakuMsgJson, "contentTopic")->text_value;

    if (strcmp(contentTopic, "/example/1/default/rfc26") == 0)
    {
      char *msg = utils_extract_wakumessage_from_signal(wakuMsgJson);

      // Decode a message using asymmetric encryption
      char *decodedMsg = NULL;
      waku_decode_asymmetric(msg, bobPrivKey, on_response, (void *)&decodedMsg);

      const nx_json *dataJson = nx_json_parse(decodedMsg, 0);

      const char *pubkey = nx_json_get(dataJson, "pubkey")->text_value;
      const char *base64data = nx_json_get(dataJson, "data")->text_value;

      size_t data_len = b64_decoded_size(base64data);
      unsigned char *data = malloc(data_len);

      b64_decode(base64data, data, data_len);

      printf(">>> Received \"%s\" from %s\n", data, pubkey);

      free(msg);
      free(decodedMsg);
      free(data);

      fflush(stdout);
    }
  }

  nx_json_free(json);
}

int main(int argc, char *argv[])
{
  // configJSON can be NULL too to use defaults. Any value not defined will have
  // a default set
  char *configJSON = "{\"host\": \"0.0.0.0\", \"port\": 60000, "
                     "\"logLevel\":\"error\", \"store\":true}";
  void* ctx = waku_new(configJSON, on_error, NULL);

    // Set callback to be executed each time a message is received
  waku_set_event_callback(ctx, callBack);

  // Start the node, enabling the waku protocols
  waku_start(ctx, on_error, NULL);

  // Obtain the node's peerID
  char *peerID = NULL;
  waku_peerid(ctx, on_response, (void *)&peerID);
  printf("PeerID: %s\n", peerID);

  // Obtain the node's multiaddresses
  char *addresses = NULL;
  waku_listen_addresses(ctx, on_response, (void *)&addresses);
  printf("Addresses: %s\n", addresses);

  // Build a content topic
  char *contentTopic = NULL;
  waku_content_topic("example", "1", "default", "rfc26", on_response,
                     (void *)&contentTopic);
  printf("Content Topic: %s\n", contentTopic);

  // Obtain the default pubsub topic
  char *defaultPubsubTopic = NULL;
  waku_default_pubsub_topic(on_response, (void *)&defaultPubsubTopic);
  printf("Default pubsub topic: %s\n", defaultPubsubTopic);

  // To use dns discovery, and retrieve nodes from a enrtree url
  char *discoveredNodes = NULL;
  waku_dns_discovery(ctx, "enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im",
                     "", 0, on_response, (void *)&discoveredNodes);
  printf("Discovered nodes: %s\n", discoveredNodes);

  // Connect to a node
  waku_connect(ctx, "/dns4/node-01.do-ams3.waku.test.status.im/tcp/30303/"
               "p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W",
               0, on_response, NULL);

  // To see a store query in action:
  // char query[1000];
  // sprintf(query,
  //        "{\"pubsubTopic\":\"%s\", \"pagingOptions\":{\"pageSize\": 40,  "
  //        "\"forward\":false}}",
  //        pubsubTopic);
  // char *query_result = NULL;
  // waku_store_query(query, NULL, 0, on_response, (void*)&query_result);
  // printf("%s\n", query_result);
  char contentFilter[1000];
  sprintf(contentFilter,
          "{\"pubsubTopic\":\"%s\",\"contentTopics\":[\"%s\"]}",
          defaultPubsubTopic, contentTopic);
  waku_relay_subscribe(ctx, contentFilter, on_error, NULL);

  int i = 0;
  int version = 1;
  while (i < 5)
  {
    i++;

    char wakuMsg[1000];
    unsigned char plain_text[] = "Hello World!";
    char *msgPayload = b64_encode(&plain_text[0], 12);

    // Build the waku message
    sprintf(wakuMsg,
            "{\"payload\":\"%s\",\"contentTopic\":\"%s\",\"timestamp\":%" PRIu64
            "}",
            msgPayload, contentTopic, nowInNanosecs());
    free(msgPayload);

    // Use asymmetric encryption to encrypt the waku message
    char *encodedMessage = NULL;
    waku_encode_asymmetric(wakuMsg, bobPubKey, alicePrivKey, on_response,
                           (void *)&encodedMessage);

    // Broadcast via waku relay
    char *messageID = NULL;
    waku_relay_publish(ctx, encodedMessage, defaultPubsubTopic, 0, on_response,
                       (void *)&messageID);
    printf("MessageID: %s\n", messageID);

    sleep(1);
  }

  // To retrieve messages from local store, set store:true in the node
  // config, and use waku_store_local_query
  // char query2[1000];
  // sprintf(query2,
  //        "{\"pubsubTopic\":\"%s\", \"pagingOptions\":{\"pageSize\":  40, "
  //        "\"forward\":false}}",
  //        pubsubTopic);
  // char *local_result = NULL;
  // waku_store_local_query(query2, on_response, (void*)&local_result);
  // printf("%s\n", local_result);

  // Stop the node's execution
  waku_stop(ctx, on_response, NULL);

  // Release resources allocated to waku
  waku_free(ctx, on_response, NULL);

  // TODO: free all char*

  return 0;
}
