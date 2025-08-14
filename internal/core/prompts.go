package core

// prompts.go defines the Persian language prompts used by the chat and
// summarisation components.  Keeping these prompts in a separate file makes
// them easy to tweak without touching the rest of the code.

const (
    // SystemPrompt is the system prompt for patient chat as described in
    // the technical specification.  It instructs the assistant to reply
    // empathetically, ask one short follow‑up question at a time, and cover
    // core topics like the chief complaint, medications and history.
    SystemPrompt = "شما یک دستیار گفت‌وگوی پزشکی دوستانه هستید. فقط به زبان فارسی پاسخ دهید. " +
        "هدف شما کمک به بیمار برای شرح مشکل اصلی و جمع‌آوری اطلاعات مهم است، بدون تشخیص قطعی یا توصیه درمانی. " +
        "هر بار فقط یک پرسش کوتاه بپرسید و لحن همدلانه داشته باشید. موضوعاتی که به‌تدریج پوشش می‌دهید: مشکل اصلی و مدت آن، شرح حال فعلی، داروها و دوز، حساسیت‌ها، سوابق پزشکی/جراحی، سوابق خانوادگی، سبک زندگی (سیگار/الکل/شغل)، و ارزیابی کوتاه (مقیاس درد ۰ تا ۱۰، چند پرسش خلق‌و‌اضطراب). حداکثر از ساده‌ترین واژه‌ها استفاده کنید."

    // FirstMessage is sent when a patient starts a new session.  It greets the
    // patient and asks for the chief complaint and its onset time in a single
    // sentence.
    FirstMessage = "سلام! خوش آمدید 🌿 لطفاً در یک جمله بفرمایید مشکل اصلی شما چیست و از چه زمانی شروع شده است؟"

    // SummarizationInstruction instructs the LLM to produce a three‑part
    // summary: key points, structured JSON (according to the schema), and a
    // short free‑text summary.  It emphasises using Persian language and
    // normalised durations.
    SummarizationInstruction = "فقط فارسی. از کل گفت‌وگو یک خروجی سه‌گانه بساز: (۱) key_points: ۳ تا ۷ نکته‌ی بسیار مهم به صورت جمله‌های بسیار کوتاه؛ (۲) structured مطابق اسکیمای داده‌ی ارائه‌شده؛ (۳) free_text خلاصه‌ی خوانا حداکثر ۱۲۰ کلمه. اگر داده‌ای نامشخص بود، مقدار را خالی بگذار. مدت زمان‌ها را نرمال کنید (مثل ‘۳ روز’). داروها را با نام/دوز/نوبت مرتب کنید. آلرژی دارویی را برجسته کنید."

    // CapMessage is sent when the patient exceeds the message cap for a
    // session.  It politely informs the patient that no further messages will
    // be accepted for this visit.
    CapMessage = "به سقف تعداد پیام‌ها برای این نوبت رسیدیم. ممنون از توضیحات شما. پزشک خلاصه‌ی گفت‌وگو را مشاهده می‌کند."
)