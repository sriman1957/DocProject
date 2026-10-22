import 'package:flutter/material.dart';

import '../../auth/models/user.dart';

class FacultyDashboardScreen extends StatelessWidget {
  const FacultyDashboardScreen({
    required this.user,
    required this.onNavigateTab,
    required this.onLogout,
    super.key,
  });

  final User user;
  final ValueChanged<int> onNavigateTab;
  final VoidCallback onLogout;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    final facultyBadgeText =
        user.facultyCode != null && user.facultyCode!.isNotEmpty
        ? 'Faculty • ${user.facultyCode}'
        : 'Faculty';

    return SingleChildScrollView(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 500),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Welcome Header Card
              Card(
                elevation: 0,
                color: colors.primaryContainer.withValues(alpha: 0.4),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(16),
                  side: BorderSide(
                    color: colors.primary.withValues(alpha: 0.15),
                  ),
                ),
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Icon(
                            Icons.co_present_rounded,
                            size: 32,
                            color: colors.primary,
                          ),
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 12,
                              vertical: 4,
                            ),
                            decoration: BoxDecoration(
                              color: colors.primary,
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Text(
                              facultyBadgeText,
                              style: theme.textTheme.labelMedium?.copyWith(
                                color: colors.onPrimary,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 16),
                      Text(
                        'Welcome, ${user.fullName}',
                        style: theme.textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        user.email,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: colors.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 24),

              Text(
                'Collection Management',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 12),

              // Action Tile 1: Groups Management
              _FacultyActionTile(
                icon: Icons.groups_outlined,
                title: 'Groups Management',
                subtitle: 'Supervise batch groups and review student rosters',
                onTap: () => onNavigateTab(1),
              ),
              const SizedBox(height: 12),

              // Action Tile 2: Subgroups & Portfolios
              _FacultyActionTile(
                icon: Icons.folder_copy_outlined,
                title: 'Subgroup Activities',
                subtitle: 'Organize document collections and configure access windows',
                onTap: () => onNavigateTab(2),
              ),
              const SizedBox(height: 12),

              // Action Tile 3: Notifications
              _FacultyActionTile(
                icon: Icons.notifications_outlined,
                title: 'Activity Notifications',
                subtitle: 'Receive updates on student document submissions',
                onTap: () => onNavigateTab(3),
              ),
              const SizedBox(height: 28),

              Text(
                'Faculty Account Details',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 12),

              // Faculty Account Details Card
              Card(
                elevation: 0,
                color: colors.surfaceContainerLow,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(14),
                  side: BorderSide(
                    color: colors.outlineVariant.withValues(alpha: 0.4),
                  ),
                ),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 14,
                  ),
                  child: Column(
                    children: [
                      _FacultyDetailRow(
                        label: 'Faculty Name',
                        value: user.fullName,
                      ),
                      const Divider(height: 16),
                      _FacultyDetailRow(
                        label: 'College ID',
                        value: user.collegeId.toString(),
                      ),
                      if (user.facultyCode != null &&
                          user.facultyCode!.isNotEmpty) ...[
                        const Divider(height: 16),
                        _FacultyDetailRow(
                          label: 'Faculty Code',
                          value: user.facultyCode!,
                        ),
                      ],
                      const Divider(height: 16),
                      _FacultyDetailRow(label: 'Email', value: user.email),
                      const Divider(height: 16),
                      const _FacultyDetailRow(
                        label: 'Role',
                        value: 'Faculty Member',
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 20),

              // Sign Out Button
              OutlinedButton.icon(
                onPressed: onLogout,
                icon: const Icon(Icons.logout),
                label: const Text('Sign out'),
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size.fromHeight(48),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _FacultyActionTile extends StatelessWidget {
  const _FacultyActionTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Card(
      elevation: 0,
      color: colors.surfaceContainerLow,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(color: colors.outlineVariant.withValues(alpha: 0.4)),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(14),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: colors.primaryContainer.withValues(alpha: 0.5),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Icon(icon, color: colors.primary, size: 24),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Icon(Icons.chevron_right_rounded, color: colors.onSurfaceVariant),
            ],
          ),
        ),
      ),
    );
  }
}

class _FacultyDetailRow extends StatelessWidget {
  const _FacultyDetailRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          label,
          style: theme.textTheme.bodySmall?.copyWith(
            color: colors.onSurfaceVariant,
          ),
        ),
        Text(
          value,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w500,
          ),
        ),
      ],
    );
  }
}
