// Arabic translations — Phase 24.2 sovereign / government market support.
//
// Translations are professional-grade SOC terminology, not literal.
// E.g. "Pop out" → "إخراج إلى نافذة" (extract to window) rather than the
// nonsensical literal "ظهور للخارج".
//
// Keep keys in sync with en.ts. Missing keys fall back to English with a
// dev-mode console warning (see ./index.ts).

import type { Translations } from './index';

export const ar: Translations = {
  // ── Common
  'common.cancel': 'إلغاء',
  'common.confirm': 'تأكيد',
  'common.save': 'حفظ',
  'common.close': 'إغلاق',
  'common.delete': 'حذف',
  'common.search': 'بحث',
  'common.loading': 'جارٍ التحميل…',
  'common.error': 'خطأ',
  'common.success': 'تم بنجاح',
  'common.warning': 'تحذير',
  'common.info': 'معلومة',
  'common.retry': 'إعادة المحاولة',
  'common.dismiss': 'إخفاء',

  // ── App chrome
  'titlebar.search.placeholder': 'البحث في الأوامر…',
  'titlebar.operator.label': 'المشغّل',
  'titlebar.popouts': '{0} نوافذ منبثقة',
  'titlebar.popouts.singular': '{0} نافذة منبثقة',

  // ── Notifications
  'notifications.empty.title': 'لا توجد إشعارات بعد',
  'notifications.empty.description': 'تظهر هنا التنبيهات وأحداث النظام والتحذيرات.',
  'notifications.markAllRead': 'تعليم الكل كمقروء',
  'notifications.clearAll': 'مسح الكل',
  'notifications.unreadBadge': '{0} جديد',
  'notifications.title': 'الإشعارات',

  // ── Pop-out
  'popout.button': 'إخراج',
  'popout.button.tooltip': 'إخراج إلى نافذة جديدة — اسحبها إلى شاشة أخرى',
  'popout.failed.title': 'فشل الإخراج',
  'popout.unavailable.title': 'الإخراج غير متاح',

  // ── Setup wizard
  'setup.title': 'الإعداد الأولي',
  'setup.step.admin': 'حساب المسؤول',
  'setup.step.alertChannel': 'قناة التنبيه',
  'setup.step.detectionPack': 'حزمة الكشف',
  'setup.step.tutorial': 'تم بنجاح',
  'setup.admin.email': 'البريد الإلكتروني',
  'setup.admin.passphrase': 'عبارة المرور (12 حرفًا على الأقل)',
  'setup.admin.passphraseConfirm': 'تأكيد عبارة المرور',
  'setup.continue': 'متابعة',
  'setup.back': 'رجوع',
  'setup.finish': 'إنهاء الإعداد',
  'setup.initialising': 'جارٍ التهيئة…',

  // ── Status / health
  'status.healthy': 'سليم',
  'status.degraded': 'متراجع',
  'status.critical': 'حرج',
  'status.unknown': 'غير معروف',
  'health.degraded.banner': 'الأنبوب تحت الضغط — أداء متراجع، الاستيعاب ما زال طبيعيًا.',
  'health.critical.banner': 'توقف الأنبوب — المخزن قريب من السعة، قد تُفقد الأحداث.',

  // ── Operator banner (terminal)
  'operator.alerts.label': 'تنبيه واحد على {1}',
  'operator.alerts.label.plural': '{0} تنبيهات على {1}',
  'operator.viewEvents': 'عرض الأحداث',
  'operator.isolate': 'عزل',

  // ── Sessions
  'session.restore.prompt': 'استعادة {0} جلسات سابقة؟',
  'session.restore.action': 'استعادة',
  'session.restoring': 'جارٍ الاستعادة…',

  // ── Empty / error states
  'empty.alerts.title': 'لا توجد تنبيهات في هذا العرض',
  'empty.search.title': 'لا توجد نتائج',
  'error.generic.title': 'حدث خطأ ما',

  // ── Settings
  'settings.language': 'اللغة',
  'settings.language.description': 'لغة عرض واجهة OBLIVRA.',
};
