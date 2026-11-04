#ifndef MAIN_H
#define MAIN_H

#include <stdbool.h>
#include <time.h>
#include <stdint.h>
#include "nxjson.c"

/// Convert seconds to nanoseconds
#define SEC_TO_NS(sec) ((sec)*1000000000)

#define WAKU_CALL(call)                                                        \
do {                                                                           \
  int ret = call;                                                              \
  if (ret != 0) {                                                              \
    printf("Failed the call to: %s. Returned code: %d\n", #call, ret);         \
    exit(1);                                                                   \
  }                                                                            \
} while (0)

uint64_t nowInNanosecs(){
  uint64_t nanoseconds;
  struct timespec ts;
  int return_code = timespec_get(&ts, TIME_UTC);
  if (return_code == 0)
  {
      printf("Failed to obtain timestamp.\n");
      nanoseconds = UINT64_MAX; // use this to indicate error
  }
  else
  {
      // `ts` now contains your timestamp in seconds and nanoseconds! To 
      // convert the whole struct to nanoseconds, do this:
      nanoseconds = SEC_TO_NS((uint64_t)ts.tv_sec) + (uint64_t)ts.tv_nsec;
  }
  return nanoseconds;
}


bool isError(char *input)
{
  char *jsonStr = malloc(strlen(input) + 1);
  strcpy(jsonStr, input);
  const nx_json *json = nx_json_parse(jsonStr, 0);
  bool result = false;
  if (json)
  {
    const char *errTxt = nx_json_get(json, "error")->text_value;
    result = errTxt != NULL;
    if (result)
    {
      printf("ERROR: %s\n", errTxt);
    }
  }
  nx_json_free(json);
  free(jsonStr);
  return result;
}


char *utils_extract_wakumessage_from_signal(const nx_json *wakuMsgJson)
{
    const char *payload = nx_json_get(wakuMsgJson, "payload")->text_value;
    const char *contentTopic = nx_json_get(wakuMsgJson, "contentTopic")->text_value;
    int version = nx_json_get(wakuMsgJson, "version")->int_value;
    long long timestamp = nx_json_get(wakuMsgJson, "timestamp")->int_value;
    char wakuMsg[6000];
    sprintf(wakuMsg, "{\"payload\":\"%s\",\"contentTopic\":\"%s\",\"timestamp\":%lld, \"version\":%d}", payload, contentTopic, timestamp, version);
    char *response = (char *)malloc(sizeof(char) * (strlen(wakuMsg) + 1));
    strcpy(response, wakuMsg);
    return response;
}


#endif /* MAIN_H */