import 'package:flutter/material.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../../auth/models/user.dart';
import '../../documents/controllers/document_upload_controller.dart';
import '../../subgroups/screens/subgroups_listing_screen.dart';
import '../../subgroups/services/subgroups_service.dart';
import '../../subgroups/subgroups_controller.dart';
import '../models/group.dart';

class GroupDetailsScreen extends StatelessWidget {
  const GroupDetailsScreen({
    required this.group,
    required this.user,
    this.subgroupsController,
    this.uploadController,
    super.key,
  });

  final Group group;
  final User user;
  final SubgroupsController? subgroupsController;
  final DocumentUploadController? uploadController;

  void _openSubgroups(BuildContext context) {
    final effectiveController =
        subgroupsController ??
        SubgroupsController(
          subgroupsService: SubgroupsService(
            apiClient: ApiClient(),
            tokenStorage: TokenStorage(),
          ),
        );

    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => SubgroupsListingScreen(
          parentGroup: group,
          user: user,
          subgroupsController: effectiveController,
          uploadController: uploadController,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final isStudent = user.userRole.isStudent;

    return Scaffold(
      appBar: AppBar(title: const Text('Group Details'), centerTitle: false),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 550),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Main Group Banner Card
                  Card(
                    elevation: 0,
                    color: colors.primaryContainer.withValues(alpha: 0.35),
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
                              Container(
                                width: 48,
                                height: 48,
                                decoration: BoxDecoration(
                                  color: colors.primary,
                                  borderRadius: BorderRadius.circular(12),
                                ),
                                child: Icon(
                                  Icons.groups_rounded,
                                  color: colors.onPrimary,
                                  size: 28,
                                ),
                              ),
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 10,
                                  vertical: 4,
                                ),
                                decoration: BoxDecoration(
                                  color: group.isActive
                                      ? Colors.green.withValues(alpha: 0.15)
                                      : colors.errorContainer,
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: Text(
                                  group.isActive
                                      ? 'Active Batch'
                                      : 'Inactive Batch',
                                  style: theme.textTheme.labelMedium?.copyWith(
                                    color: group.isActive
                                        ? Colors.green.shade800
                                        : colors.onErrorContainer,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 16),
                          Text(
                            group.name,
                            style: theme.textTheme.headlineSmall?.copyWith(
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          const SizedBox(height: 6),
                          Text(
                            'Group ID: #${group.id} • Branch #${group.branchId}',
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: colors.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),

                  // Action to navigate into Subgroups
                  FilledButton.icon(
                    onPressed: () => _openSubgroups(context),
                    icon: const Icon(Icons.folder_copy_outlined),
                    label: const Text('View Collection Subgroups'),
                    style: FilledButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 14),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),

                  // Description Card
                  Text(
                    'Description',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
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
                      padding: const EdgeInsets.all(16),
                      child: Text(
                        group.description.isNotEmpty ? group.description : 'No description provided for this academic group.',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: group.description.isNotEmpty
                              ? colors.onSurface
                              : colors.outline,
                          fontStyle: group.description.isNotEmpty
                              ? FontStyle.normal
                              : FontStyle.italic,
                          height: 1.4,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),

                  // Metadata Details Card
                  Text(
                    'Group Metadata',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
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
                        vertical: 12,
                      ),
                      child: Column(
                        children: [
                          _DetailRow(label: 'Group ID', value: '#${group.id}'),
                          const Divider(height: 16),
                          _DetailRow(
                            label: 'Branch ID',
                            value: '#${group.branchId}',
                          ),
                          const Divider(height: 16),
                          _DetailRow(
                            label: 'College ID',
                            value: '#${group.collegeId}',
                          ),
                          const Divider(height: 16),
                          _DetailRow(
                            label: 'Created By (User ID)',
                            value: '#${group.createdBy}',
                          ),
                          if (group.createdAt != null &&
                              group.createdAt!.isNotEmpty) ...[
                            const Divider(height: 16),
                            _DetailRow(
                              label: 'Created Date',
                              value: group.createdAt!,
                            ),
                          ],
                          if (group.updatedAt != null &&
                              group.updatedAt!.isNotEmpty) ...[
                            const Divider(height: 16),
                            _DetailRow(
                              label: 'Last Updated',
                              value: group.updatedAt!,
                            ),
                          ],
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),

                  // Role-Aware Next Phases Notice
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: colors.secondaryContainer.withValues(alpha: 0.35),
                      borderRadius: BorderRadius.circular(14),
                      border: Border.all(
                        color: colors.secondary.withValues(alpha: 0.2),
                      ),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(
                          Icons.info_outline_rounded,
                          color: colors.secondary,
                          size: 22,
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                isStudent
                                    ? 'Document Collection Activities'
                                    : 'Subgroups & Roster Management',
                                style: theme.textTheme.titleSmall?.copyWith(
                                  fontWeight: FontWeight.bold,
                                  color: colors.onSecondaryContainer,
                                ),
                              ),
                              const SizedBox(height: 4),
                              Text(
                                isStudent
                                    ? 'Subgroups and certificate submission requests for this batch will appear here as faculty create them.'
                                    : 'Creating collection subgroups and managing student member enrollments will be available in the upcoming feature phase.',
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: colors.onSecondaryContainer,
                                  height: 1.35,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _DetailRow extends StatelessWidget {
  const _DetailRow({required this.label, required this.value});

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
          style: theme.textTheme.bodyMedium?.copyWith(
            color: colors.onSurfaceVariant,
          ),
        ),
        Text(
          value,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );
  }
}
