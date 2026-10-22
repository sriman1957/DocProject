import 'package:flutter/material.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../../auth/models/user.dart';
import '../../documents/controllers/document_upload_controller.dart';
import '../../documents/services/documents_service.dart';
import '../../documents/widgets/file_upload_sheet.dart';
import '../../groups/models/group.dart';
import '../models/subgroup.dart';

class SubgroupDetailsScreen extends StatelessWidget {
  const SubgroupDetailsScreen({
    required this.subgroup,
    required this.parentGroup,
    required this.user,
    this.uploadController,
    super.key,
  });

  final Subgroup subgroup;
  final Group parentGroup;
  final User user;
  final DocumentUploadController? uploadController;

  void _openSubmissionSheet(BuildContext context) {
    final effectiveController =
        uploadController ??
        DocumentUploadController(
          documentsService: DocumentsService(
            apiClient: ApiClient(),
            tokenStorage: TokenStorage(),
          ),
        );

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => FileUploadSheet(
        uploadController: effectiveController,
        destination: UploadDestination.subgroup,
        groupId: subgroup.groupId,
        subgroupId: subgroup.id,
        subgroupName: subgroup.name,
        onSuccess: () {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(
                'Document successfully submitted to ${subgroup.name}.',
              ),
              backgroundColor: Colors.green.shade800,
            ),
          );
        },
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final isStudent = user.userRole.isStudent;

    return Scaffold(
      appBar: AppBar(title: const Text('Subgroup Details'), centerTitle: false),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 550),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Main Subgroup Header Card
                  Card(
                    elevation: 0,
                    color: colors.secondaryContainer.withValues(alpha: 0.35),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(16),
                      side: BorderSide(
                        color: colors.secondary.withValues(alpha: 0.2),
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
                                  color: colors.secondary,
                                  borderRadius: BorderRadius.circular(12),
                                ),
                                child: Icon(
                                  Icons.folder_copy_rounded,
                                  color: colors.onSecondary,
                                  size: 26,
                                ),
                              ),
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 10,
                                  vertical: 4,
                                ),
                                decoration: BoxDecoration(
                                  color: subgroup.isActive
                                      ? Colors.green.withValues(alpha: 0.15)
                                      : colors.errorContainer,
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: Text(
                                  subgroup.isActive
                                      ? 'Active Subgroup'
                                      : 'Inactive Subgroup',
                                  style: theme.textTheme.labelMedium?.copyWith(
                                    color: subgroup.isActive
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
                            subgroup.name,
                            style: theme.textTheme.headlineSmall?.copyWith(
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          const SizedBox(height: 6),
                          Row(
                            children: [
                              Icon(
                                Icons.groups_outlined,
                                size: 16,
                                color: colors.onSurfaceVariant,
                              ),
                              const SizedBox(width: 6),
                              Expanded(
                                child: Text(
                                  'Parent Group: ${parentGroup.name} (#${parentGroup.id})',
                                  style: theme.textTheme.bodyMedium?.copyWith(
                                    color: colors.onSurfaceVariant,
                                    fontWeight: FontWeight.w500,
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),

                  // Description Section
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
                        subgroup.description.isNotEmpty ? subgroup.description : 'No description provided for this collection subgroup.',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: subgroup.description.isNotEmpty
                              ? colors.onSurface
                              : colors.outline,
                          fontStyle: subgroup.description.isNotEmpty
                              ? FontStyle.normal
                              : FontStyle.italic,
                          height: 1.4,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),

                  // Metadata Details Section
                  Text(
                    'Subgroup Metadata',
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
                          _DetailRow(
                            label: 'Subgroup ID',
                            value: '#${subgroup.id}',
                          ),
                          const Divider(height: 16),
                          _DetailRow(
                            label: 'Parent Group ID',
                            value: '#${subgroup.groupId}',
                          ),
                          const Divider(height: 16),
                          _DetailRow(
                            label: 'College ID',
                            value: '#${subgroup.collegeId}',
                          ),
                          const Divider(height: 16),
                          _DetailRow(
                            label: 'Created By (User ID)',
                            value: '#${subgroup.createdBy}',
                          ),
                          if (subgroup.createdAt != null &&
                              subgroup.createdAt!.isNotEmpty) ...[
                            const Divider(height: 16),
                            _DetailRow(
                              label: 'Created Date',
                              value: subgroup.createdAt!,
                            ),
                          ],
                          if (subgroup.updatedAt != null &&
                              subgroup.updatedAt!.isNotEmpty) ...[
                            const Divider(height: 16),
                            _DetailRow(
                              label: 'Last Updated',
                              value: subgroup.updatedAt!,
                            ),
                          ],
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),

                  // Role-Tailored Upcoming Phase Banner
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: colors.primaryContainer.withValues(alpha: 0.35),
                      borderRadius: BorderRadius.circular(14),
                      border: Border.all(
                        color: colors.primary.withValues(alpha: 0.2),
                      ),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(
                          Icons.info_outline_rounded,
                          color: colors.primary,
                          size: 22,
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                isStudent
                                    ? 'Document Submission Window'
                                    : 'Collection & Access Management',
                                style: theme.textTheme.titleSmall?.copyWith(
                                  fontWeight: FontWeight.bold,
                                  color: colors.onPrimaryContainer,
                                ),
                              ),
                              const SizedBox(height: 4),
                              Text(
                                isStudent
                                    ? 'Submit required academic certificates or files to this collection subgroup. Uploading academic certificates into this subgroup is verified against active access periods.'
                                    : 'Configuring access periods and reviewing submitted student documents for this subgroup will be enabled in subsequent features.',
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: colors.onPrimaryContainer,
                                  height: 1.35,
                                ),
                              ),
                              if (isStudent) ...[
                                const SizedBox(height: 12),
                                FilledButton.icon(
                                  key: const Key(
                                    'submit_subgroup_document_button',
                                  ),
                                  onPressed: () =>
                                      _openSubmissionSheet(context),
                                  icon: const Icon(
                                    Icons.send_rounded,
                                    size: 18,
                                  ),
                                  label: const Text('Submit Document'),
                                ),
                              ],
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
