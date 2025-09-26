package main

/*
   // —— 不需要任何 #include 或 -I —— //
   // 声明要用到的 C 符号（用最通用的签名，避免依赖 enum/typedef）：

   // ggml 的日志回调设置：一直存在
   extern void ggml_log_set(void (*log_cb)(int, const char*, void*), void*);

   // whisper 的日志回调设置：不同版本可能没有。用弱符号防止链接报错
   void whisper_log_set(void (*log_cb)(int, const char*, void*), void*) __attribute__((weak));

   // 空日志回调
   static void no_log(int level, const char * text, void * user_data) {
       (void)level; (void)text; (void)user_data;
   }

   static void disable_native_logs() {
       // 如果当前链接的库里导出了 whisper_log_set，就调用；否则跳过
       if (whisper_log_set) {
           whisper_log_set(no_log, 0);
       }
       // ggml 的总开关
       ggml_log_set(no_log, 0);
   }
*/
import "C"

func DisableNativeLogs() {
	C.disable_native_logs()
}
