// English translations — the source-of-truth locale. Other locale files
// must keep their key sets in sync with this one.
//
// Naming convention: `<area>.<element>.<state>` lowercase, dot-separated.
// Use `{0}`, `{1}` for positional interpolation.

import type { Translations } from './index';

export const en: Translations = {
  // ── Common
  'common.cancel': 'Cancel',
  'common.confirm': 'Confirm',
  'common.save': 'Save',
  'common.close': 'Close',
  'common.delete': 'Delete',
  'common.search': 'Search',
  'common.loading': 'Loading…',
  'common.error': 'Error',
  'common.success': 'Success',
  'common.warning': 'Warning',
  'common.info': 'Info',
  'common.retry': 'Retry',
  'common.dismiss': 'Dismiss',

  // ── App chrome
  'titlebar.search.placeholder': 'Search commands…',
  'titlebar.operator.label': 'OPERATOR',
  'titlebar.popouts': '{0} POP-OUTS',
  'titlebar.popouts.singular': '{0} POP-OUT',

  // ── Notifications
  'notifications.empty.title': 'No notifications yet',
  'notifications.empty.description': 'Alerts, system events, and warnings appear here.',
  'notifications.markAllRead': 'Mark all read',
  'notifications.clearAll': 'Clear all',
  'notifications.unreadBadge': '{0} NEW',
  'notifications.title': 'Notifications',

  // ── Pop-out
  'popout.button': 'Pop out',
  'popout.button.tooltip': 'Pop out to a new window — drag to another monitor',
  'popout.failed.title': 'Pop-out failed',
  'popout.unavailable.title': 'Pop-out unavailable',

  // ── Setup wizard
  'setup.title': 'Initial Setup',
  'setup.step.admin': 'Administrator Account',
  'setup.step.alertChannel': 'Alert Channel',
  'setup.step.detectionPack': 'Detection Pack',
  'setup.step.tutorial': "You're done",
  'setup.admin.email': 'Email',
  'setup.admin.passphrase': 'Passphrase (12+ characters)',
  'setup.admin.passphraseConfirm': 'Confirm passphrase',
  'setup.continue': 'Continue',
  'setup.back': 'Back',
  'setup.finish': 'Finish setup',
  'setup.initialising': 'Initialising…',

  // ── Status / health
  'status.healthy': 'Healthy',
  'status.degraded': 'Degraded',
  'status.critical': 'Critical',
  'status.unknown': 'Unknown',
  'health.degraded.banner': 'Pipeline under load — degraded performance, ingestion still nominal.',
  'health.critical.banner': 'Pipeline stalled — buffer near capacity, events may be dropped.',

  // ── Operator banner (terminal)
  'operator.alerts.label': '{0} alert on {1}',
  'operator.alerts.label.plural': '{0} alerts on {1}',
  'operator.viewEvents': 'View events',
  'operator.isolate': 'Isolate',

  // ── Sessions
  'session.restore.prompt': 'Restore {0} previous sessions?',
  'session.restore.action': 'Restore',
  'session.restoring': 'Restoring…',

  // ── Empty / error states
  'empty.alerts.title': 'No alerts in this view',
  'empty.search.title': 'No results',
  'error.generic.title': 'Something went wrong',

  // ── Settings
  'settings.language': 'Language',
  'settings.language.description': 'Display language for the OBLIVRA interface.',
};
