class Group {
  const Group({
    required this.id,
    required this.collegeId,
    required this.branchId,
    required this.name,
    this.description = '',
    required this.createdBy,
    this.isActive = true,
    this.createdAt,
    this.updatedAt,
  });

  final int id;
  final int collegeId;
  final int branchId;
  final String name;
  final String description;
  final int createdBy;
  final bool isActive;
  final String? createdAt;
  final String? updatedAt;

  factory Group.fromJson(Map<String, dynamic> json) {
    return Group(
      id: (json['id'] as num).toInt(),
      collegeId: (json['college_id'] as num).toInt(),
      branchId: (json['branch_id'] as num).toInt(),
      name: json['name'] as String? ?? '',
      description: json['description'] as String? ?? '',
      createdBy: (json['created_by'] as num).toInt(),
      isActive: json['is_active'] as bool? ?? true,
      createdAt: json['created_at']?.toString(),
      updatedAt: json['updated_at']?.toString(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'college_id': collegeId,
      'branch_id': branchId,
      'name': name,
      'description': description,
      'created_by': createdBy,
      'is_active': isActive,
      if (createdAt != null) 'created_at': createdAt,
      if (updatedAt != null) 'updated_at': updatedAt,
    };
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Group &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          collegeId == other.collegeId &&
          branchId == other.branchId &&
          name == other.name &&
          description == other.description &&
          createdBy == other.createdBy &&
          isActive == other.isActive;

  @override
  int get hashCode =>
      id.hashCode ^
      collegeId.hashCode ^
      branchId.hashCode ^
      name.hashCode ^
      description.hashCode ^
      createdBy.hashCode ^
      isActive.hashCode;
}
