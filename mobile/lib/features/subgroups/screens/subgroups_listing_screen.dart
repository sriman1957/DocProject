import 'package:flutter/material.dart';

import '../../auth/models/user.dart';
import '../../groups/models/group.dart';
import '../models/subgroup.dart';
import '../subgroups_controller.dart';
import '../widgets/subgroup_card.dart';
import 'subgroup_details_screen.dart';

class SubgroupsListingScreen extends StatefulWidget {
  const SubgroupsListingScreen({
    required this.parentGroup,
    required this.user,
    required this.subgroupsController,
    this.autoLoad = true,
    super.key,
  });

  final Group parentGroup;
  final User user;
  final SubgroupsController subgroupsController;
  final bool autoLoad;

  @override
  State<SubgroupsListingScreen> createState() => _SubgroupsListingScreenState();
}

class _SubgroupsListingScreenState extends State<SubgroupsListingScreen> {
  @override
  void initState() {
    super.initState();
    if (widget.autoLoad) {
      widget.subgroupsController.loadSubgroups(widget.parentGroup.id);
    }
  }

  void _openSubgroupDetails(Subgroup subgroup) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => SubgroupDetailsScreen(
          subgroup: subgroup,
          parentGroup: widget.parentGroup,
          user: widget.user,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final isStudent = widget.user.userRole.isStudent;

    return Scaffold(
      appBar: AppBar(title: Text(widget.parentGroup.name), centerTitle: false),
      body: SafeArea(
        child: ListenableBuilder(
          listenable: widget.subgroupsController,
          builder: (context, _) {
            final state = widget.subgroupsController.state;

            return RefreshIndicator(
              onRefresh: () =>
                  widget.subgroupsController.refresh(widget.parentGroup.id),
              child: switch (state) {
                SubgroupsInitial() => const SizedBox.shrink(),
                SubgroupsLoading() => const Center(
                  child: Padding(
                    padding: EdgeInsets.all(32),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        CircularProgressIndicator(),
                        SizedBox(height: 16),
                        Text('Loading subgroups...'),
                      ],
                    ),
                  ),
                ),
                SubgroupsError(:final message, :final canRetry) => Center(
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
                          'Failed to Load Subgroups',
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
                                widget.subgroupsController.loadSubgroups(
                                  widget.parentGroup.id,
                                  force: true,
                                ),
                            icon: const Icon(Icons.refresh_rounded),
                            label: const Text('Try Again'),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
                SubgroupsEmpty() => Center(
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
                            Icons.folder_copy_outlined,
                            size: 36,
                            color: colors.onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          'No Subgroups Available',
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          isStudent
                              ? 'No document collection activities have been created for this group yet. Check back once faculty configure collection windows.'
                              : 'No collection subgroups found in this group. Activities configured for this batch will appear here.',
                          textAlign: TextAlign.center,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: colors.onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 20),
                        OutlinedButton.icon(
                          onPressed: () => widget.subgroupsController.refresh(
                            widget.parentGroup.id,
                          ),
                          icon: const Icon(Icons.refresh_rounded),
                          label: const Text('Refresh'),
                        ),
                      ],
                    ),
                  ),
                ),
                SubgroupsLoaded(:final subgroups) => ListView.separated(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.symmetric(
                    horizontal: 20,
                    vertical: 20,
                  ),
                  itemCount: subgroups.length + 1,
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
                            Container(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 10,
                                vertical: 4,
                              ),
                              decoration: BoxDecoration(
                                color: colors.primaryContainer.withValues(
                                  alpha: 0.5,
                                ),
                                borderRadius: BorderRadius.circular(8),
                              ),
                              child: Text(
                                'Parent Group: ${widget.parentGroup.name} (#${widget.parentGroup.id})',
                                style: theme.textTheme.labelMedium?.copyWith(
                                  color: colors.onPrimaryContainer,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ),
                            const SizedBox(height: 12),
                            Text(
                              'Collection Subgroups',
                              style: theme.textTheme.titleLarge?.copyWith(
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              isStudent
                                  ? 'Select an activity to view collection details and requirements.'
                                  : 'Supervise document collection subgroups for this batch.',
                              style: theme.textTheme.bodyMedium?.copyWith(
                                color: colors.onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      );
                    }

                    final subgroup = subgroups[index - 1];
                    return SubgroupCard(
                      subgroup: subgroup,
                      onTap: () => _openSubgroupDetails(subgroup),
                    );
                  },
                ),
              },
            );
          },
        ),
      ),
    );
  }
}
