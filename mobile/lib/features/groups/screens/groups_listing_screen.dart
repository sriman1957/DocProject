import 'package:flutter/material.dart';

import '../../access_periods/controllers/access_periods_controller.dart';
import '../../auth/models/user.dart';
import '../../documents/controllers/document_upload_controller.dart';
import '../../documents/controllers/subgroup_submissions_controller.dart';
import '../../subgroups/subgroups_controller.dart';
import '../groups_controller.dart';
import '../models/group.dart';
import '../widgets/group_card.dart';
import 'group_details_screen.dart';

class GroupsListingScreen extends StatefulWidget {
  const GroupsListingScreen({
    required this.user,
    required this.groupsController,
    this.subgroupsController,
    this.uploadController,
    this.accessPeriodsController,
    this.submissionsController,
    this.autoLoad = true,
    super.key,
  });

  final User user;
  final GroupsController groupsController;
  final SubgroupsController? subgroupsController;
  final DocumentUploadController? uploadController;
  final AccessPeriodsController? accessPeriodsController;
  final SubgroupSubmissionsController? submissionsController;
  final bool autoLoad;

  @override
  State<GroupsListingScreen> createState() => _GroupsListingScreenState();
}

class _GroupsListingScreenState extends State<GroupsListingScreen> {
  @override
  void initState() {
    super.initState();
    if (widget.autoLoad) {
      widget.groupsController.loadGroups();
    }
  }

  void _openGroupDetails(Group group) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => GroupDetailsScreen(
          group: group,
          user: widget.user,
          subgroupsController: widget.subgroupsController,
          uploadController: widget.uploadController,
          accessPeriodsController: widget.accessPeriodsController,
          submissionsController: widget.submissionsController,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final isStudent = widget.user.userRole.isStudent;

    return ListenableBuilder(
      listenable: widget.groupsController,
      builder: (context, _) {
        final state = widget.groupsController.state;

        return RefreshIndicator(
          onRefresh: widget.groupsController.refresh,
          child: switch (state) {
            GroupsInitial() => const SizedBox.shrink(),
            GroupsLoading() => const Center(
              child: Padding(
                padding: EdgeInsets.all(32),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    CircularProgressIndicator(),
                    SizedBox(height: 16),
                    Text('Loading groups...'),
                  ],
                ),
              ),
            ),
            GroupsError(:final message, :final canRetry) => Center(
              child: SingleChildScrollView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(
                      Icons.error_outline_rounded,
                      size: 56,
                      color: colors.error,
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'Failed to Load Groups',
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      message,
                      textAlign: TextAlign.center,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                    if (canRetry) ...[
                      const SizedBox(height: 20),
                      FilledButton.tonalIcon(
                        onPressed: () =>
                            widget.groupsController.loadGroups(force: true),
                        icon: const Icon(Icons.refresh_rounded),
                        label: const Text('Try Again'),
                      ),
                    ],
                  ],
                ),
              ),
            ),
            GroupsEmpty() => Center(
              child: SingleChildScrollView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Container(
                      width: 72,
                      height: 72,
                      decoration: BoxDecoration(
                        color: colors.surfaceContainerHighest,
                        shape: BoxShape.circle,
                      ),
                      child: Icon(
                        Icons.groups_outlined,
                        size: 36,
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'No Groups Found',
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      isStudent
                          ? 'You are not enrolled in any academic groups yet. Once faculty or administrators add you to a batch, it will appear here.'
                          : 'No batch groups found in your assigned branches. Check back later or contact your college administrator.',
                      textAlign: TextAlign.center,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                    const SizedBox(height: 20),
                    OutlinedButton.icon(
                      onPressed: widget.groupsController.refresh,
                      icon: const Icon(Icons.refresh_rounded),
                      label: const Text('Refresh'),
                    ),
                  ],
                ),
              ),
            ),
            GroupsLoaded(:final groups) => ListView.separated(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 20),
              itemCount: groups.length + 1,
              separatorBuilder: (_, index) => index == 0
                  ? const SizedBox.shrink()
                  : const SizedBox(height: 12),
              itemBuilder: (context, index) {
                if (index == 0) {
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          isStudent
                              ? 'Enrolled Academic Groups'
                              : 'Assigned Batch Groups',
                          style: theme.textTheme.titleLarge?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          isStudent
                              ? 'Select a group to view details and submission access.'
                              : 'Select a group to inspect branch and batch configuration.',
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: colors.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  );
                }

                final group = groups[index - 1];
                return GroupCard(
                  group: group,
                  onTap: () => _openGroupDetails(group),
                );
              },
            ),
          },
        );
      },
    );
  }
}
