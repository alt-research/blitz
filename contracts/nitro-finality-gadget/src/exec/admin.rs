use babylon_bindings::BabylonMsg;
use cosmwasm_std::{DepsMut, MessageInfo, Event, Response};

use crate::{
    error::ContractError,
    state::{config::{ADMIN, CONFIG, IS_ENABLED}, finality::{BLOCK_HASHES, BLOCK_VOTES, EVIDENCES, SIGNATURES}}
};

// Enable or disable the finality gadget.
// Only callable by contract admin.
// If disabled, the verifier should bypass the EOTS verification logic, allowing the OP derivation
// derivation pipeline to pass through. Note this should be implemented in the verifier and is not
// enforced by the contract itself.
pub fn set_enabled(
    deps: DepsMut,
    info: MessageInfo,
    enabled: bool,
) -> Result<Response<BabylonMsg>, ContractError> {
    // Check caller is admin
    check_admin(&deps, info)?;
    // Check if the finality gadget is already in the desired state
    if IS_ENABLED.load(deps.storage)? == enabled {
        if enabled {
            return Err(ContractError::AlreadyEnabled {});
        } else {
            return Err(ContractError::AlreadyDisabled {});
        }
    }
    // Disable finality gadget
    IS_ENABLED.save(deps.storage, &enabled)?;
    Result::Ok(Response::default())
}

// Helper function to check caller is contract admin
fn check_admin(deps: &DepsMut, info: MessageInfo) -> Result<(), ContractError> {
    // Check caller is admin
    if !ADMIN.is_admin(deps.as_ref(), &info.sender)? {
        return Err(ContractError::Unauthorized {});
    }
    Ok(())
}

// Reset finality gadget, ONLY for test
pub fn reset(
    deps: DepsMut,
    info: MessageInfo
) -> Result<Response<BabylonMsg>, ContractError> {
    // Check caller is admin
    check_admin(&deps, info)?;

    // Reset all storages
    SIGNATURES.clear(deps.storage);
    BLOCK_HASHES.clear(deps.storage);
    BLOCK_VOTES.clear(deps.storage);
    EVIDENCES.clear(deps.storage);

    let res = Response::default()
        .add_event( Event::new("admin_reset"));

    Result::Ok(res)
}

// Set Commit 
pub fn set_commit_block_height_interval(
    deps: DepsMut,
    info: MessageInfo,
    commit_block_height_interval: u64
) -> Result<Response<BabylonMsg>, ContractError> {
    // Check caller is admin
    check_admin(&deps, info)?;

    // Reset all storages
    let mut config = CONFIG.load(deps.storage)?;
    let old_commit_block_height_interval = config.commit_block_height_interval;
    config.commit_block_height_interval = commit_block_height_interval;

    CONFIG.save(deps.storage, &config)?;


    let res = Response::default()
        .add_event(
             Event::new("admin_set_commit_block_height_interval").
             add_attribute("old_commit_block_height_interval", old_commit_block_height_interval.to_string()).
             add_attribute("commit_block_height_interval", commit_block_height_interval.to_string())
        );

    Result::Ok(res)
}