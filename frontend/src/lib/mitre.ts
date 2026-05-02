// Static MITRE ATT&CK Enterprise matrix subset — tactics with their
// most-detected techniques. Keeps the dashboard self-contained (no
// network fetch) and air-gap-friendly. The list is intentionally not
// exhaustive: it covers the techniques OBLIVRA's builtin and Sigma
// rule packs actually emit, which is what an operator wants to see in
// "what coverage do I have".
//
// Updating: when we ship rules with new technique IDs, append them
// under the right tactic. Source for canonical names:
// https://attack.mitre.org/matrices/enterprise/

export interface MitreTactic {
  id: string;
  name: string;
  // technique IDs in the order they should display under the tactic
  techniques: { id: string; name: string }[];
}

export const MITRE_TACTICS: MitreTactic[] = [
  {
    id: 'TA0001', name: 'Initial Access',
    techniques: [
      { id: 'T1078',     name: 'Valid Accounts' },
      { id: 'T1190',     name: 'Exploit Public-Facing App' },
      { id: 'T1133',     name: 'External Remote Services' },
      { id: 'T1566',     name: 'Phishing' },
    ],
  },
  {
    id: 'TA0002', name: 'Execution',
    techniques: [
      { id: 'T1059',     name: 'Command and Scripting Interpreter' },
      { id: 'T1059.001', name: 'PowerShell' },
      { id: 'T1059.003', name: 'Windows Command Shell' },
      { id: 'T1059.004', name: 'Unix Shell' },
      { id: 'T1218',     name: 'System Binary Proxy Execution' },
      { id: 'T1106',     name: 'Native API' },
    ],
  },
  {
    id: 'TA0003', name: 'Persistence',
    techniques: [
      { id: 'T1053',     name: 'Scheduled Task/Job' },
      { id: 'T1136',     name: 'Create Account' },
      { id: 'T1098',     name: 'Account Manipulation' },
      { id: 'T1547',     name: 'Boot or Logon Autostart' },
      { id: 'T1543',     name: 'Create or Modify System Process' },
      { id: 'T1574',     name: 'Hijack Execution Flow' },
    ],
  },
  {
    id: 'TA0004', name: 'Privilege Escalation',
    techniques: [
      { id: 'T1068',     name: 'Exploitation for Privilege Escalation' },
      { id: 'T1548',     name: 'Abuse Elevation Control' },
      { id: 'T1611',     name: 'Escape to Host' },
    ],
  },
  {
    id: 'TA0005', name: 'Defense Evasion',
    techniques: [
      { id: 'T1027',     name: 'Obfuscated Files / Information' },
      { id: 'T1070',     name: 'Indicator Removal' },
      { id: 'T1070.001', name: 'Clear Windows Event Logs' },
      { id: 'T1070.003', name: 'Clear Command History' },
      { id: 'T1070.006', name: 'Timestomp' },
      { id: 'T1490',     name: 'Inhibit System Recovery' },
      { id: 'T1562',     name: 'Impair Defenses' },
      { id: 'T1562.001', name: 'Disable / Modify Tools' },
      { id: 'T1562.006', name: 'Indicator Blocking' },
      { id: 'T1218',     name: 'Signed Binary Proxy Execution' },
    ],
  },
  {
    id: 'TA0006', name: 'Credential Access',
    techniques: [
      { id: 'T1003',     name: 'OS Credential Dumping' },
      { id: 'T1003.001', name: 'LSASS Memory' },
      { id: 'T1110',     name: 'Brute Force' },
      { id: 'T1110.001', name: 'Password Guessing' },
      { id: 'T1110.003', name: 'Password Spraying' },
      { id: 'T1555',     name: 'Credentials from Password Stores' },
      { id: 'T1552',     name: 'Unsecured Credentials' },
    ],
  },
  {
    id: 'TA0007', name: 'Discovery',
    techniques: [
      { id: 'T1087',     name: 'Account Discovery' },
      { id: 'T1018',     name: 'Remote System Discovery' },
      { id: 'T1083',     name: 'File / Directory Discovery' },
      { id: 'T1057',     name: 'Process Discovery' },
      { id: 'T1082',     name: 'System Information Discovery' },
    ],
  },
  {
    id: 'TA0008', name: 'Lateral Movement',
    techniques: [
      { id: 'T1021',     name: 'Remote Services' },
      { id: 'T1021.001', name: 'RDP' },
      { id: 'T1021.002', name: 'SMB / Admin Shares' },
      { id: 'T1021.004', name: 'SSH' },
      { id: 'T1570',     name: 'Lateral Tool Transfer' },
    ],
  },
  {
    id: 'TA0009', name: 'Collection',
    techniques: [
      { id: 'T1005',     name: 'Data from Local System' },
      { id: 'T1039',     name: 'Data from Network Shared Drive' },
      { id: 'T1056',     name: 'Input Capture' },
    ],
  },
  {
    id: 'TA0011', name: 'Command and Control',
    techniques: [
      { id: 'T1071',     name: 'Application Layer Protocol' },
      { id: 'T1071.004', name: 'DNS' },
      { id: 'T1105',     name: 'Ingress Tool Transfer' },
      { id: 'T1095',     name: 'Non-Application Layer Protocol' },
    ],
  },
  {
    id: 'TA0010', name: 'Exfiltration',
    techniques: [
      { id: 'T1041',     name: 'Exfil over C2 Channel' },
      { id: 'T1048',     name: 'Exfil over Alternative Protocol' },
    ],
  },
  {
    id: 'TA0040', name: 'Impact',
    techniques: [
      { id: 'T1485',     name: 'Data Destruction' },
      { id: 'T1486',     name: 'Data Encrypted for Impact' },
      { id: 'T1490',     name: 'Inhibit System Recovery' },
      { id: 'T1489',     name: 'Service Stop' },
    ],
  },
];
